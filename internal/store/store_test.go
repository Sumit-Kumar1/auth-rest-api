package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"auth-rest-api/internal/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/container"
	gofrHTTP "gofr.dev/pkg/gofr/http"
)

var errRedis = errors.New("redis error")

const testFailStr = "TEST[%d] Failed - %s"

func newTestContext(t *testing.T) (*gofr.Context, *container.Mocks) {
	t.Helper()

	mockContainer, mocks := container.NewMockContainer(t)

	return &gofr.Context{
		Context:   context.Background(),
		Container: mockContainer,
	}, mocks
}

// helper to create a StatusCmd with a value
func statusCmdOK() *redis.StatusCmd {
	cmd := redis.NewStatusCmd(context.Background())
	cmd.SetVal("OK")

	return cmd
}

// helper to create a StatusCmd with an error
func statusCmdErr(err error) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(context.Background())
	cmd.SetErr(err)

	return cmd
}

// helper to create an IntCmd with a value
func intCmdVal(val int64) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(val)

	return cmd
}

// helper to create an IntCmd with an error
func intCmdErr(err error) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetErr(err)

	return cmd
}

// helper to create a StringCmd with a value
func stringCmdVal(val string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetVal(val)

	return cmd
}

// helper to create a StringCmd with an error
func stringCmdErr(err error) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(err)

	return cmd
}

// helper to create a MapStringStringCmd with a value
func mapCmdVal(val map[string]string) *redis.MapStringStringCmd {
	cmd := redis.NewMapStringStringCmd(context.Background())
	cmd.SetVal(val)

	return cmd
}

// helper to create a BoolCmd with a value
func boolCmdVal(val bool) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	cmd.SetVal(val)

	return cmd
}

func TestStore_CreateUser(t *testing.T) {
	userEmail := "test@example.com"
	userID := uuid.NewString()
	passwd := []byte("hashedpassword")
	s := New()

	tests := []struct {
		name     string
		user     *models.UserData
		mockCall func(mocks *container.Mocks)
		wantErr  error
	}{
		{
			name: "success",
			user: &models.UserData{ID: userID, Email: userEmail, Password: passwd},
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Set(context.Background(), emaild+userEmail, userID, time.Duration(0)).
					Return(statusCmdOK())
				mocks.Redis.EXPECT().HSet(context.Background(), userd+userID,
					map[string]any{"id": userID, "email": userEmail, "password": passwd}).
					Return(intCmdVal(1))
			},
		},
		{
			name: "email set redis nil",
			user: &models.UserData{ID: userID, Email: userEmail, Password: passwd},
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Set(context.Background(), emaild+userEmail, userID, time.Duration(0)).
					Return(statusCmdErr(redis.Nil))
			},
			wantErr: gofrHTTP.ErrorEntityAlreadyExist{},
		},
		{
			name: "email set redis error",
			user: &models.UserData{ID: userID, Email: userEmail, Password: passwd},
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Set(context.Background(), emaild+userEmail, userID, time.Duration(0)).
					Return(statusCmdErr(errRedis))
			},
			wantErr: errRedis,
		},
		{
			name: "hset redis nil",
			user: &models.UserData{ID: userID, Email: userEmail, Password: passwd},
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Set(context.Background(), emaild+userEmail, userID, time.Duration(0)).
					Return(statusCmdOK())
				mocks.Redis.EXPECT().HSet(context.Background(), userd+userID,
					map[string]any{"id": userID, "email": userEmail, "password": passwd}).
					Return(intCmdErr(redis.Nil))
			},
			wantErr: gofrHTTP.ErrorEntityAlreadyExist{},
		},
		{
			name: "hset redis error",
			user: &models.UserData{ID: userID, Email: userEmail, Password: passwd},
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Set(context.Background(), emaild+userEmail, userID, time.Duration(0)).
					Return(statusCmdOK())
				mocks.Redis.EXPECT().HSet(context.Background(), userd+userID,
					map[string]any{"id": userID, "email": userEmail, "password": passwd}).
					Return(intCmdErr(errRedis))
			},
			wantErr: errRedis,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mocks := newTestContext(t)
			tt.mockCall(mocks)

			err := s.CreateUser(c, tt.user)
			assert.Equalf(t, tt.wantErr, err, testFailStr, i, tt.name)
		})
	}
}

func TestStore_GetUserByEmail(t *testing.T) {
	userEmail := "test@example.com"
	userID := uuid.NewString()
	passwd := "hashedpass"
	s := New()

	tests := []struct {
		name     string
		email    string
		mockCall func(mocks *container.Mocks)
		want     *models.UserData
		wantErr  error
	}{
		{
			name:  "success",
			email: userEmail,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Get(context.Background(), emaild+userEmail).
					Return(stringCmdVal(userID))
				mocks.Redis.EXPECT().HGetAll(context.Background(), userd+userID).
					Return(mapCmdVal(map[string]string{"id": userID, "email": userEmail, "password": passwd}))
			},
			want: &models.UserData{ID: userID, Email: userEmail, Password: []byte(passwd)},
		},
		{
			name:  "email not found",
			email: userEmail,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Get(context.Background(), emaild+userEmail).
					Return(stringCmdErr(redis.Nil))
			},
			wantErr: gofrHTTP.ErrorEntityNotFound{Name: "user", Value: userEmail},
		},
		{
			name:  "get email redis error",
			email: userEmail,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Get(context.Background(), emaild+userEmail).
					Return(stringCmdErr(errRedis))
			},
			wantErr: errRedis,
		},
		{
			name:  "hgetall redis error",
			email: userEmail,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Get(context.Background(), emaild+userEmail).
					Return(stringCmdVal(userID))
				mocks.Redis.EXPECT().HGetAll(context.Background(), userd+userID).
					Return(mapCmdVal(map[string]string{}))
			},
			wantErr: gofrHTTP.ErrorEntityNotFound{Name: "user", Value: userEmail},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mocks := newTestContext(t)
			tt.mockCall(mocks)

			got, err := s.GetUserByEmail(c, tt.email)
			assert.Equalf(t, tt.wantErr, err, testFailStr, i, tt.name)
			assert.Equalf(t, tt.want, got, testFailStr, i, tt.name)
		})
	}
}

func TestStore_IsTokenRevoked(t *testing.T) {
	tokenID := uuid.NewString()
	s := New()

	tests := []struct {
		name     string
		tokenID  string
		mockCall func(mocks *container.Mocks)
		want     bool
		wantErr  error
	}{
		{
			name:    "token not revoked",
			tokenID: tokenID,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), accTokend+tokenID).
					Return(intCmdVal(1))
			},
			want: false,
		},
		{
			name:    "token revoked - not found",
			tokenID: tokenID,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), accTokend+tokenID).
					Return(intCmdVal(0))
			},
			want: true,
		},
		{
			name:    "redis nil error - revoked",
			tokenID: tokenID,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), accTokend+tokenID).
					Return(intCmdErr(redis.Nil))
			},
			want: true,
		},
		{
			name:    "redis error",
			tokenID: tokenID,
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), accTokend+tokenID).
					Return(intCmdErr(errRedis))
			},
			want:    false,
			wantErr: errRedis,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mocks := newTestContext(t)
			tt.mockCall(mocks)

			got, err := s.IsTokenRevoked(c, tt.tokenID)
			assert.Equalf(t, tt.wantErr, err, testFailStr, i, tt.name)
			assert.Equalf(t, tt.want, got, testFailStr, i, tt.name)
		})
	}
}

func TestStore_IncrementFailedLogin(t *testing.T) {
	userEmail := "test@example.com"
	key := failedLoginAttempts + userEmail
	s := New()

	tests := []struct {
		name     string
		mockCall func(mocks *container.Mocks)
		want     int
		wantErr  error
	}{
		{
			name: "first failed attempt",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Incr(context.Background(), key).Return(intCmdVal(1))
				mocks.Redis.EXPECT().Expire(context.Background(), key, failedLoginTTL).Return(boolCmdVal(true))
			},
			want: 1,
		},
		{
			name: "subsequent attempt - no expire",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Incr(context.Background(), key).Return(intCmdVal(3))
			},
			want: 3,
		},
		{
			name: "max attempts reached - locks account",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Incr(context.Background(), key).Return(intCmdVal(5))
				mocks.Redis.EXPECT().Set(context.Background(), accountLocked+userEmail, "locked", failedLoginTTL).
					Return(statusCmdOK())
			},
			want: 5,
		},
		{
			name: "incr redis error",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Incr(context.Background(), key).Return(intCmdErr(errRedis))
			},
			wantErr: errRedis,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mocks := newTestContext(t)
			tt.mockCall(mocks)

			got, err := s.IncrementFailedLogin(c, userEmail)
			assert.Equalf(t, tt.wantErr, err, testFailStr, i, tt.name)

			if tt.wantErr == nil {
				assert.Equalf(t, tt.want, got, testFailStr, i, tt.name)
			}
		})
	}
}

func TestStore_ResetFailedLogin(t *testing.T) {
	userEmail := "test@example.com"
	key := failedLoginAttempts + userEmail
	s := New()

	tests := []struct {
		name     string
		mockCall func(mocks *container.Mocks)
		wantErr  error
	}{
		{
			name: "success",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Del(context.Background(), key).Return(intCmdVal(1))
			},
		},
		{
			name: "redis error",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Del(context.Background(), key).Return(intCmdErr(errRedis))
			},
			wantErr: errRedis,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mocks := newTestContext(t)
			tt.mockCall(mocks)

			err := s.ResetFailedLogin(c, userEmail)
			assert.Equalf(t, tt.wantErr, err, testFailStr, i, tt.name)
		})
	}
}

func TestStore_IsAccountLocked(t *testing.T) {
	userEmail := "test@example.com"
	key := accountLocked + userEmail
	s := New()

	tests := []struct {
		name     string
		mockCall func(mocks *container.Mocks)
		want     bool
		wantErr  error
	}{
		{
			name: "account locked",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), key).Return(intCmdVal(1))
			},
			want: true,
		},
		{
			name: "account not locked",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), key).Return(intCmdVal(0))
			},
			want: false,
		},
		{
			name: "redis nil - not locked",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), key).Return(intCmdErr(redis.Nil))
			},
			want: false,
		},
		{
			name: "redis error",
			mockCall: func(mocks *container.Mocks) {
				mocks.Redis.EXPECT().Exists(context.Background(), key).Return(intCmdErr(errRedis))
			},
			want:    false,
			wantErr: errRedis,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mocks := newTestContext(t)
			tt.mockCall(mocks)

			got, err := s.IsAccountLocked(c, userEmail)
			assert.Equalf(t, tt.wantErr, err, testFailStr, i, tt.name)
			assert.Equalf(t, tt.want, got, testFailStr, i, tt.name)
		})
	}
}
