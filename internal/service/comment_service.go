package service

import (
	"context"
	"errors"

	"medcon/internal/models"
	"medcon/internal/repository"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	postRepo    *repository.PostRepository
	userRepo    *repository.UserRepository
}

func NewCommentService(commentRepo *repository.CommentRepository, postRepo *repository.PostRepository, userRepo *repository.UserRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
	}
}

func (s *CommentService) CreateComment(ctx context.Context, postID, authorID string, req *models.CreateCommentRequest) (*models.Comment, error) {
	// Verify post exists
	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}

	// Verify author exists
	author, err := s.userRepo.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, errors.New("author not found")
	}

	comment := models.NewComment(postID, authorID, req.Content)

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, errors.New("failed to create comment")
	}

	// Increment reply count on post
	if err := s.postRepo.IncrementReplyCount(ctx, postID); err != nil {
		// Log error but don't fail the request
	}

	comment.Author = author
	return comment, nil
}

func (s *CommentService) ListComments(ctx context.Context, postID string, page, limit int) ([]*models.Comment, int64, error) {
	comments, total, err := s.commentRepo.ListByPostID(ctx, postID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Populate authors
	for _, comment := range comments {
		author, err := s.userRepo.GetByID(ctx, comment.AuthorID)
		if err == nil && author != nil {
			author.PasswordHash = ""
			comment.Author = author
		}
	}

	return comments, total, nil
}

func (s *CommentService) AcceptComment(ctx context.Context, commentID, postAuthorID string) (*models.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, errors.New("comment not found")
	}

	// Verify the requester is the post author
	post, err := s.postRepo.GetByID(ctx, comment.PostID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}

	if post.AuthorID != postAuthorID {
		return nil, errors.New("only the post author can accept comments")
	}

	// Unaccept any previously accepted comment for this post
	// (In a real app, you might want to handle this differently)

	// Accept this comment
	if err := s.commentRepo.Update(ctx, commentID, map[string]interface{}{"is_accepted": true}); err != nil {
		return nil, errors.New("failed to accept comment")
	}

	// Update post status to answered
	if err := s.postRepo.Update(ctx, post.ID, map[string]interface{}{"status": models.PostStatusAnswered}); err != nil {
		// Log error
	}

	comment.IsAccepted = true
	return comment, nil
}
