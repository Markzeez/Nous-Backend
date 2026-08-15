package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"medcon/internal/auth"
	"medcon/internal/db"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub manages all WebSocket connections and rooms.
type Hub struct {
	mu          sync.RWMutex
	clients     map[string]*Client          // userId -> client
	rooms       map[string]map[string]bool   // roomId -> set of userIds
	db          *db.Client
	jwtSecret   string
}

type Client struct {
	UserID   string
	Role     string
	Conn     *websocket.Conn
	Hub      *Hub
	SendChan chan []byte
}

type WSMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

func NewHub(database *db.Client, jwtSecret string) *Hub {
	return &Hub{
		clients:   make(map[string]*Client),
		rooms:     make(map[string]map[string]bool),
		db:        database,
		jwtSecret: jwtSecret,
	}
}

// ServeWS handles the WebSocket upgrade. Token is passed as a query parameter.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ValidateToken(tokenStr, h.jwtSecret)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] Upgrade error:", err)
		return
	}

	client := &Client{
		UserID:   claims.ID,
		Role:     string(claims.Role),
		Conn:     conn,
		Hub:      h,
		SendChan: make(chan []byte, 256),
	}

	h.mu.Lock()
	h.clients[claims.ID] = client
	h.mu.Unlock()

	log.Printf("[WS] Connected: %s (%s)", claims.ID, claims.Role)

	go client.writePump()
	go client.readPump()
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for msg := range c.SendChan {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.removeClient(c)
		c.Conn.Close()
	}()
	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var msg WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		c.Hub.handleEvent(c, msg)
	}
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c.UserID)
	for roomID, members := range h.rooms {
		delete(members, c.UserID)
		if len(members) == 0 {
			delete(h.rooms, roomID)
		}
	}
	close(c.SendChan)
	log.Printf("[WS] Disconnected: %s", c.UserID)
}

func (h *Hub) handleEvent(sender *Client, msg WSMessage) {
	switch msg.Event {
	case "room:join":
		var d struct {
			RoomID string `json:"roomId"`
		}
		json.Unmarshal(msg.Data, &d)
		h.mu.Lock()
		if h.rooms[d.RoomID] == nil {
			h.rooms[d.RoomID] = make(map[string]bool)
		}
		h.rooms[d.RoomID][sender.UserID] = true
		h.mu.Unlock()
		h.broadcastToRoom(d.RoomID, "room:user_joined", map[string]string{
			"userId": sender.UserID, "role": sender.Role,
		})

	case "room:leave":
		var d struct {
			RoomID string `json:"roomId"`
		}
		json.Unmarshal(msg.Data, &d)
		h.mu.Lock()
		if members, ok := h.rooms[d.RoomID]; ok {
			delete(members, sender.UserID)
			if len(members) == 0 {
				delete(h.rooms, d.RoomID)
			}
		}
		h.mu.Unlock()
		h.broadcastToRoom(d.RoomID, "room:user_left", map[string]string{"userId": sender.UserID})

	case "message:send":
		var d struct {
			RoomID  string `json:"roomId"`
			Message struct {
				Content string `json:"content"`
				Type    string `json:"type"`
			} `json:"message"`
		}
		json.Unmarshal(msg.Data, &d)
		if d.Message.Type == "" {
			d.Message.Type = "text"
		}

		row := map[string]interface{}{
			"room_id":   d.RoomID,
			"sender_id": sender.UserID,
			"content":   d.Message.Content,
			"type":      d.Message.Type,
		}
		saved, err := h.db.Insert("messages", row, "*,sender:users!messages_sender_id_fkey(id,name,role,avatar_url)")
		if err != nil {
			log.Println("[WS] Message persist error:", err)
			return
		}
		h.broadcastToRoomRaw(d.RoomID, "message:receive", saved)

	case "typing:start":
		var d struct {
			RoomID string `json:"roomId"`
			Name   string `json:"name"`
		}
		json.Unmarshal(msg.Data, &d)
		h.broadcastToRoomExcept(d.RoomID, sender.UserID, "typing:update", map[string]interface{}{
			"userId": sender.UserID, "name": d.Name, "isTyping": true,
		})

	case "typing:stop":
		var d struct {
			RoomID string `json:"roomId"`
		}
		json.Unmarshal(msg.Data, &d)
		h.broadcastToRoomExcept(d.RoomID, sender.UserID, "typing:update", map[string]interface{}{
			"userId": sender.UserID, "isTyping": false,
		})

	case "ambulance:dispatch":
		var d struct {
			Location struct {
				Lat     float64 `json:"lat"`
				Lng     float64 `json:"lng"`
				Address string  `json:"address"`
			} `json:"location"`
			Severity string `json:"severity"`
			RoomID   string `json:"roomId"`
		}
		json.Unmarshal(msg.Data, &d)

		evt := map[string]interface{}{
			"id":               "amb_" + sender.UserID,
			"patientId":        sender.UserID,
			"location":         d.Location,
			"severity":         d.Severity,
			"status":           "dispatched",
			"estimatedArrival": "8-12 minutes",
		}
		h.broadcastAll("ambulance:dispatched", evt)
		if d.RoomID != "" {
			h.broadcastToRoom(d.RoomID, "ambulance:en_route", evt)
		}

	case "booking:created":
		var d struct {
			ProfessionalID string          `json:"professionalId"`
			Booking        json.RawMessage `json:"booking"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.ProfessionalID, "booking:new_request", d.Booking)

	case "booking:accepted":
		var d struct {
			PatientID string          `json:"patientId"`
			Booking   json.RawMessage `json:"booking"`
			RoomID    string          `json:"roomId"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.PatientID, "booking:confirmed", map[string]interface{}{
			"booking": d.Booking, "roomId": d.RoomID,
		})

	case "booking:declined":
		var d struct {
			PatientID string `json:"patientId"`
			Reason    string `json:"reason"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.PatientID, "booking:declined", map[string]string{"reason": d.Reason})

	// ─── Video Call Signaling ────────────────────────────────────────────────

	case "video:call":
		var d struct {
			RoomID     string `json:"roomId"`
			TargetUser string `json:"targetUserId"`
			CallerName string `json:"callerName"`
			HasVideo   bool   `json:"hasVideo"`
			HasAudio   bool   `json:"hasAudio"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.TargetUser, "video:incoming", map[string]interface{}{
			"roomId":     d.RoomID,
			"callerId":   sender.UserID,
			"callerName": d.CallerName,
			"callerRole": sender.Role,
			"hasVideo":   d.HasVideo,
			"hasAudio":   d.HasAudio,
		})

	case "video:call_accepted":
		var d struct {
			RoomID     string `json:"roomId"`
			CallerID   string `json:"callerId"`
			AccepterID string `json:"accepterId"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.CallerID, "video:call_accepted", map[string]interface{}{
			"roomId":     d.RoomID,
			"accepterId": d.AccepterID,
		})

	case "video:call_declined":
		var d struct {
			RoomID   string `json:"roomId"`
			CallerID string `json:"callerId"`
			Reason   string `json:"reason"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.CallerID, "video:call_declined", map[string]interface{}{
			"roomId": d.RoomID,
			"reason": d.Reason,
		})

	case "video:offer":
		var d struct {
			RoomID     string                 `json:"roomId"`
			TargetUser string                 `json:"targetUserId"`
			Offer      map[string]interface{} `json:"offer"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.TargetUser, "video:offer", map[string]interface{}{
			"roomId":   d.RoomID,
			"senderId": sender.UserID,
			"offer":    d.Offer,
		})

	case "video:answer":
		var d struct {
			RoomID     string                 `json:"roomId"`
			TargetUser string                 `json:"targetUserId"`
			Answer     map[string]interface{} `json:"answer"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.TargetUser, "video:answer", map[string]interface{}{
			"roomId":   d.RoomID,
			"senderId": sender.UserID,
			"answer":   d.Answer,
		})

	case "video:ice_candidate":
		var d struct {
			RoomID     string                 `json:"roomId"`
			TargetUser string                 `json:"targetUserId"`
			Candidate  map[string]interface{} `json:"candidate"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.TargetUser, "video:ice_candidate", map[string]interface{}{
			"roomId":    d.RoomID,
			"senderId":  sender.UserID,
			"candidate": d.Candidate,
		})

	case "video:hangup":
		var d struct {
			RoomID     string `json:"roomId"`
			TargetUser string `json:"targetUserId"`
		}
		json.Unmarshal(msg.Data, &d)
		h.sendToUser(d.TargetUser, "video:hangup", map[string]interface{}{
			"roomId":   d.RoomID,
			"senderId": sender.UserID,
		})
	}
}

// ── broadcast helpers ──────────────────────────────────────────────────────

func (h *Hub) broadcastToRoom(roomID, event string, data interface{}) {
	payload, _ := json.Marshal(data)
	h.broadcastToRoomRaw(roomID, event, payload)
}

func (h *Hub) broadcastToRoomRaw(roomID, event string, rawData []byte) {
	msg, _ := json.Marshal(WSMessage{Event: event, Data: rawData})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for uid := range h.rooms[roomID] {
		if c, ok := h.clients[uid]; ok {
			select {
			case c.SendChan <- msg:
			default:
			}
		}
	}
}

func (h *Hub) broadcastToRoomExcept(roomID, excludeUID, event string, data interface{}) {
	payload, _ := json.Marshal(data)
	msg, _ := json.Marshal(WSMessage{Event: event, Data: payload})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for uid := range h.rooms[roomID] {
		if uid == excludeUID {
			continue
		}
		if c, ok := h.clients[uid]; ok {
			select {
			case c.SendChan <- msg:
			default:
			}
		}
	}
}

func (h *Hub) broadcastAll(event string, data interface{}) {
	payload, _ := json.Marshal(data)
	msg, _ := json.Marshal(WSMessage{Event: event, Data: payload})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.SendChan <- msg:
		default:
		}
	}
}

func (h *Hub) sendToUser(userID, event string, data interface{}) {
	payload, _ := json.Marshal(data)
	msg, _ := json.Marshal(WSMessage{Event: event, Data: payload})
	h.mu.RLock()
	defer h.mu.RUnlock()
	if c, ok := h.clients[userID]; ok {
		select {
		case c.SendChan <- msg:
		default:
		}
	}
}
