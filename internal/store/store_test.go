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

var errRedis = errors.New("redis error")

func TestStore_CreateUser(t *testing.T) {
	db, mock := redismock.NewClientMock()
	s := New(db)
	ctx := context.Background()
	email := "dummy@testmail.com"
	passwd := []byte(uuid.NewString())
	usrID := uuid.NewString()

	tests := []struct {
		name     string
		user     *models.UserData
		mockCall func()
		wantErr  error
	}{
		{
			name: "valid case",
			user: &models.UserData{ID: usrID, Email: email, Password: passwd},
			mockCall: func() {
				mock.ExpectSet(emaild+email, usrID, 0).SetVal("1")
				mock.ExpectHSet(userd+usrID, map[string]any{
					"id": usrID, "email": email, "password": passwd,
				}).SetVal(1)
			},
		},
		{
			name:     "email entry create nil",
			user:     &models.UserData{ID: usrID, Email: email, Password: passwd},
			mockCall: func() { mock.ExpectSet(emaild+email, usrID, 0).RedisNil() },
			wantErr:  models.ErrUserAlreadyExists,
		},
		{
			name:     "db error",
			user:     &models.UserData{ID: usrID, Email: email, Password: passwd},
			mockCall: func() { mock.ExpectSet(emaild+email, usrID, 0).SetErr(models.ErrDBNotConnected) },
			wantErr:  models.ErrDBNotConnected,
		},
		{
			name: "email entry found",
			user: &models.UserData{ID: usrID, Email: email, Password: passwd},
			mockCall: func() {
				mock.ExpectSet(emaild+email, usrID, 0).SetVal("0")
				mock.ExpectHSet(userd+usrID, map[string]any{
					"id": usrID, "email": email, "password": passwd}).RedisNil()
			},
			wantErr: models.ErrUserAlreadyExists,
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
	usrID := uuid.NewString()
	resp := map[string]string{
		"id": usrID, "email": email, "password": passwd,
	}

	tests := []struct {
		name     string
		email    string
		mockCall func(mock redismock.ClientMock)
		want     *models.UserData
		wantErr  error
	}{
		{
			name:  "valid case",
			email: email,
			mockCall: func(mock redismock.ClientMock) {
				mock.ExpectGet(emaild + email).SetVal(usrID)
				mock.ExpectHGetAll(userd + usrID).SetVal(resp)
			},
			want: &models.UserData{ID: usrID, Email: email, Password: []byte(passwd)},
		},
		{
			name:  "no entry for email",
			email: email,
			mockCall: func(mock redismock.ClientMock) {
				mock.ExpectGet(emaild + email).RedisNil()
			},
			wantErr: models.ErrUserNotFound,
		},
		{
			name:  "redis error",
			email: email,
			mockCall: func(mock redismock.ClientMock) {
				mock.ExpectGet(emaild + email).SetErr(models.ErrDBNotConnected)
			},
			wantErr: models.ErrDBNotConnected,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall(mock)

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
	email := "sumit@kumar.com"
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
			tokenIDs: []string{tk1, ""},
			mockCall: func() {
				mock.ExpectDel(accTokend + tk1).SetVal(1)
				mock.ExpectSRem(userAccTokend+email, tk1).SetVal(1)
			},
			wantErr: nil,
		},
		{
			name:     "valid case - 2",
			tokenIDs: []string{tk1, tk2},
			mockCall: func() {
				mock.ExpectDel(accTokend + tk1).SetVal(1)
				mock.ExpectSRem(userAccTokend+email, tk1).SetVal(1)

				mock.ExpectDel(refTokend + tk2).SetVal(1)
				mock.ExpectSRem(userRefTokend+email, tk2).SetVal(1)
			},
			wantErr: nil,
		},
		{
			name:     "redis error on delete",
			tokenIDs: []string{tk1, ""},
			mockCall: func() {
				mock.ExpectDel(accTokend + tk1).SetVal(1)
				mock.ExpectSRem(userAccTokend+email, tk1).SetErr(errRedis)
			},
			wantErr: errRedis,
		},
	}

	for i, tt := range tests {
		mock.ExpectTxPipeline()
		tt.mockCall()

		if tt.wantErr == nil {
			mock.ExpectTxPipelineExec()
		}

		assert.Equal(t, tt.wantErr, s.DeleteToken(ctx, email, tt.tokenIDs[0], tt.tokenIDs[1]),
			"TEST[%d] Failed - %s", i, tt.name)
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
				mock.ExpectExists(accTokend + revokedID).SetVal(0)
			},
			want:    true,
			wantErr: nil,
		},
		{
			name:    "token not revoked",
			tokenID: tokenID,
			mockCall: func() {
				mock.ExpectExists(accTokend + tokenID).SetVal(1)
			},
			want:    false,
			wantErr: nil,
		},
		{
			name:    "redis error on check",
			tokenID: revokedID,
			mockCall: func() {
				mock.ExpectExists(accTokend + revokedID).SetErr(errRedis)
			},
			want:    false,
			wantErr: errRedis,
		},
		{
			name:    "empty token ID",
			tokenID: "",
			mockCall: func() {
				mock.ExpectExists(accTokend).SetVal(0)
			},
			want: true,
		},
		{
			name:    "redis Nil",
			tokenID: revokedID,
			mockCall: func() {
				mock.ExpectExists(accTokend + revokedID).RedisNil()
			},
			want: true,
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
