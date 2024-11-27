package store

import (
	"context"
	"errors"
	"testing"

	"auth-rest-api/internal/models"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStore_CreateUser(t *testing.T) {
	db, mock := redismock.NewClientMock()
	s := New(db)
	ctx := context.Background()
	email := "dummy@testmail.com"
	passwd := []byte(uuid.NewString())

	tests := []struct {
		name     string
		user     *models.UserData
		mockCall func()
		wantErr  error
	}{
		{
			name:     "valid case",
			user:     &models.UserData{Email: email, Password: passwd},
			mockCall: func() { mock.ExpectHSet("users", email, passwd).SetVal(1) },
		},
		{
			name:     "email entry create nil",
			user:     &models.UserData{Email: email, Password: passwd},
			mockCall: func() { mock.ExpectHSet("users", email, passwd).RedisNil() },
			wantErr:  models.ErrUserAlreadyExists,
		},
		{
			name:     "db error",
			user:     &models.UserData{Email: email, Password: passwd},
			mockCall: func() { mock.ExpectHSet("users", email, passwd).SetErr(models.ErrDBNotConnected) },
			wantErr:  models.ErrDBNotConnected,
		},
		{
			name:     "email entry found",
			user:     &models.UserData{Email: email, Password: passwd},
			mockCall: func() { mock.ExpectHSet("users", email, passwd).SetVal(0) },
			wantErr:  models.ErrUserAlreadyExists,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			assert.Equalf(t, tt.wantErr, s.CreateUser(ctx, tt.user), "TEST[%d] Failed - %s", i, tt.name)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_GetUserByEmail(t *testing.T) {
	db, mock := redismock.NewClientMock()
	s := New(db)
	ctx := context.Background()
	email := "dummy@testmail.com"
	passwd := uuid.NewString()

	tests := []struct {
		name     string
		email    string
		mockCall func()
		want     *models.UserData
		wantErr  error
	}{
		{
			name:     "valid case",
			email:    email,
			mockCall: func() { mock.ExpectHGet("users", email).SetVal(passwd) },
			want:     &models.UserData{Email: email, Password: []byte(passwd)},
		},
		{
			name:     "empty password",
			email:    email,
			mockCall: func() { mock.ExpectHGet("users", email).SetVal("") },
			wantErr:  models.ErrNotFound("user"),
		},
		{
			name:     "no entry for email",
			email:    email,
			mockCall: func() { mock.ExpectHGet("users", email).RedisNil() },
			wantErr:  models.ErrNotFound("user"),
		},
		{
			name:     "redis error",
			email:    email,
			mockCall: func() { mock.ExpectHGet("users", email).SetErr(models.ErrDBNotConnected) },
			wantErr:  models.ErrDBNotConnected,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			got, err := s.GetUserByEmail(ctx, tt.email)
			assert.Equalf(t, tt.wantErr, err, "TEST[%d] Failed - %s", i, tt.name)
			assert.Equalf(t, tt.want, got, "TEST[%d] Failed - %s", i, tt.name)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_DeleteToken(t *testing.T) {
	db, mock := redismock.NewClientMock()
	s := New(db)
	ctx := context.Background()
	tk1 := uuid.NewString()
	tk2 := uuid.NewString()

	tests := []struct {
		name     string
		tokenIDs []string
		mockCall func()
		wantErr  error
	}{
		{
			name:     "valid case",
			tokenIDs: []string{tk1},
			mockCall: func() {
				mock.ExpectDel(tk1).SetVal(1)
			},
			wantErr: nil,
		},
		{
			name:     "valid case - 2",
			tokenIDs: []string{tk1, tk2},
			mockCall: func() {
				mock.ExpectDel(tk1, tk2).SetVal(2)
			},
			wantErr: nil,
		},
		{
			name:     "delete non-existent token",
			tokenIDs: []string{"nonexistent"},
			mockCall: func() {
				mock.ExpectDel("nonexistent").SetVal(0)
			},
			wantErr: models.NewConstError("delete error"),
		},
		{
			name:     "redis error on delete",
			tokenIDs: []string{tk1},
			mockCall: func() {
				mock.ExpectDel(tk1).SetErr(errors.New("redis error"))
			},
			wantErr: errors.New("redis error"),
		},
	}
	for i, tt := range tests {
		tt.mockCall()

		assert.Equal(t, tt.wantErr, s.DeleteToken(ctx, tt.tokenIDs...), "TEST[%d] Failed - %s", i, tt.name)
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_IsTokenRevoked(t *testing.T) {
	db, mock := redismock.NewClientMock()
	s := New(db)
	ctx := context.Background()

	revokedID := uuid.NewString()
	tokenID := uuid.NewString()

	tests := []struct {
		name     string
		tokenID  string
		mockCall func()
		want     bool
		wantErr  error
	}{
		{
			name:    "token revoked",
			tokenID: revokedID,
			mockCall: func() {
				mock.ExpectExists(revokedID).SetVal(0)
			},
			want:    true,
			wantErr: nil,
		},
		{
			name:    "token not revoked",
			tokenID: tokenID,
			mockCall: func() {
				mock.ExpectExists(tokenID).SetVal(1)
			},
			want:    false,
			wantErr: nil,
		},
		{
			name:    "redis error on check",
			tokenID: revokedID,
			mockCall: func() {
				mock.ExpectExists(revokedID).SetErr(errors.New("redis error"))
			},
			want:    false,
			wantErr: errors.New("redis error"),
		},
		{
			name:    "empty token ID",
			tokenID: "",
			mockCall: func() {
				mock.ExpectExists("").SetVal(0)
			},
			want:    true,
			wantErr: nil,
		},
		{
			name:    "redis Nil",
			tokenID: revokedID,
			mockCall: func() {
				mock.ExpectExists(revokedID).RedisNil()
			},
			want:    true,
			wantErr: nil,
		},
	}

	for i, tt := range tests {
		tt.mockCall()

		got, err := s.IsTokenRevoked(ctx, tt.tokenID)

		assert.Equalf(t, tt.wantErr, err, "TEST[%d] Failed - %s", i, tt.name)
		assert.Equalf(t, tt.want, got, "TEST[%d] Failed - %s", i, tt.name)
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}
