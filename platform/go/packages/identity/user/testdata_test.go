package user_test

import (
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/user"
)

// Test fixtures for user package.

var testUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func testActiveUser() *user.User {
	now := time.Now()
	return &user.User{
		ID:        testUserID,
		Email:     "test@example.com",
		FullName:  "Test User",
		Role:      "team-member",
		Status:    user.UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func testInvitedUser() *user.User {
	now := time.Now()
	return &user.User{
		ID:        uuid.New(),
		Email:     "invited@example.com",
		FullName:  "",
		Role:      "team-member",
		Status:    user.UserStatusInvited,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func testDeletedUser() *user.User {
	now := time.Now()
	deletedAt := now.Add(-24 * time.Hour)
	return &user.User{
		ID:        uuid.New(),
		Email:     "deleted@example.com",
		FullName:  "Deleted User",
		Role:      "team-member",
		Status:    user.UserStatusDeleted,
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: &deletedAt,
	}
}
