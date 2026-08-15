package service

import (
	"context"
	"errors"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type PostService struct {
	postRepo *repository.PostRepository
	userRepo *repository.UserRepository
}

func NewPostService(postRepo *repository.PostRepository, userRepo *repository.UserRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

func (s *PostService) CreatePost(ctx context.Context, authorID string, req *models.CreatePostRequest) (*models.Post, error) {
	// Verify author exists
	author, err := s.userRepo.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, errors.New("author not found")
	}

	category := models.PostCategory(req.Category)
	var drugName *string
	if req.DrugName != "" {
		drugName = &req.DrugName
	}

	post := models.NewPost(authorID, req.Title, req.Content, category, req.Tags, drugName)

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, errors.New("failed to create post")
	}

	post.Author = author
	return post, nil
}

func (s *PostService) GetPost(ctx context.Context, id string) (*models.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}

	// Populate author
	author, err := s.userRepo.GetByID(ctx, post.AuthorID)
	if err != nil {
		return nil, err
	}
	if author != nil {
		author.PasswordHash = ""
		post.Author = author
	}

	return post, nil
}

func (s *PostService) ListPosts(ctx context.Context, query *models.PostListQuery) ([]*models.Post, int64, error) {
	filter := make(map[string]interface{})

	if query.Category != "" {
		filter["category"] = query.Category
	}
	if query.Status != "" {
		filter["status"] = query.Status
	}
	if query.AuthorID != "" {
		filter["author_id"] = query.AuthorID
	}
	if query.Search != "" {
		filter["$or"] = []map[string]interface{}{
			{"title": map[string]string{"$regex": query.Search, "$options": "i"}},
			{"content": map[string]string{"$regex": query.Search, "$options": "i"}},
		}
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := query.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	posts, total, err := s.postRepo.List(ctx, filter, page, limit, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	// Populate authors
	for _, post := range posts {
		author, err := s.userRepo.GetByID(ctx, post.AuthorID)
		if err == nil && author != nil {
			author.PasswordHash = ""
			post.Author = author
		}
	}

	return posts, total, nil
}

func (s *PostService) UpdatePost(ctx context.Context, id, userID string, req *models.UpdatePostRequest) (*models.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}

	// Check ownership
	if post.AuthorID != userID {
		return nil, errors.New("not authorized to update this post")
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.DrugName != "" {
		updates["drug_name"] = req.DrugName
	}

	if err := s.postRepo.Update(ctx, id, updates); err != nil {
		return nil, errors.New("failed to update post")
	}

	return s.GetPost(ctx, id)
}

func (s *PostService) DeletePost(ctx context.Context, id, userID string, isAdmin bool) error {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post == nil {
		return errors.New("post not found")
	}

	// Check ownership or admin
	if post.AuthorID != userID && !isAdmin {
		return errors.New("not authorized to delete this post")
	}

	return s.postRepo.Delete(ctx, id)
}

func (s *PostService) IncrementViewCount(ctx context.Context, id string) error {
	return s.postRepo.IncrementViewCount(ctx, id)
}
