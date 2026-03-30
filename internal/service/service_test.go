package service

import (
	"errors"
	"testing"

	"auth-rest-api/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	gofrHTTP "gofr.dev/pkg/gofr/http"
	"golang.org/x/crypto/bcrypt"
)

func newTestService(t *testing.T) (*Service, *MockStorer) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockStore := NewMockStorer(ctrl)

	return New(mockStore), mockStore
}

func TestService_SignUp(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "test_access_secret")
	t.Setenv("REFRESH_SECRET", "test_refresh_secret")

	svc, mockStore := newTestService(t)

	tests := []struct {
		name     string
		user     *models.UserReq
		mockCall func()
		wantErr  bool
	}{
		{
			name: "success",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(nil, gofrHTTP.ErrorEntityNotFound{Name: "user", Value: "test@example.com"})
				mockStore.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:     "nil user",
			user:     nil,
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name:     "invalid email",
			user:     &models.UserReq{Email: "invalid", Password: "Test@1234"},
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name: "user already exists",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(&models.UserData{ID: "123", Email: "test@example.com"}, nil)
			},
			wantErr: true,
		},
		{
			name: "store GetUserByEmail error",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "store CreateUser error",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(nil, gofrHTTP.ErrorEntityNotFound{Name: "user", Value: "test@example.com"})
				mockStore.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(errors.New("create error"))
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			err := svc.SignUp(nil, tt.user)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
			}
		})
	}
}

func TestService_SignIn(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "test_access_secret")
	t.Setenv("REFRESH_SECRET", "test_refresh_secret")

	svc, mockStore := newTestService(t)

	hashedPass, _ := bcrypt.GenerateFromPassword([]byte("Test@1234"), 10)
	userID := uuid.NewString()

	tests := []struct {
		name     string
		user     *models.UserReq
		mockCall func()
		wantErr  bool
	}{
		{
			name: "success",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().IsAccountLocked(gomock.Any(), "test@example.com").Return(false, nil)
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(&models.UserData{ID: userID, Email: "test@example.com", Password: hashedPass}, nil)
				mockStore.EXPECT().ResetFailedLogin(gomock.Any(), "test@example.com").Return(nil)
				mockStore.EXPECT().CreateToken(gomock.Any(), "test@example.com", gomock.Any()).Return(nil)
			},
		},
		{
			name:     "nil user",
			user:     nil,
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name:     "invalid email",
			user:     &models.UserReq{Email: "invalid", Password: "Test@1234"},
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name: "account locked",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().IsAccountLocked(gomock.Any(), "test@example.com").Return(true, nil)
			},
			wantErr: true,
		},
		{
			name: "IsAccountLocked error",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().IsAccountLocked(gomock.Any(), "test@example.com").Return(false, errors.New("redis error"))
			},
			wantErr: true,
		},
		{
			name: "user not found",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().IsAccountLocked(gomock.Any(), "test@example.com").Return(false, nil)
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(nil, gofrHTTP.ErrorEntityNotFound{Name: "user", Value: "test@example.com"})
				mockStore.EXPECT().IncrementFailedLogin(gomock.Any(), "test@example.com").Return(1, nil)
			},
			wantErr: true,
		},
		{
			name: "wrong password",
			user: &models.UserReq{Email: "test@example.com", Password: "Wrong@1234"},
			mockCall: func() {
				mockStore.EXPECT().IsAccountLocked(gomock.Any(), "test@example.com").Return(false, nil)
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(&models.UserData{ID: userID, Email: "test@example.com", Password: hashedPass}, nil)
				mockStore.EXPECT().IncrementFailedLogin(gomock.Any(), "test@example.com").Return(1, nil)
			},
			wantErr: true,
		},
		{
			name: "create token error",
			user: &models.UserReq{Email: "test@example.com", Password: "Test@1234"},
			mockCall: func() {
				mockStore.EXPECT().IsAccountLocked(gomock.Any(), "test@example.com").Return(false, nil)
				mockStore.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").
					Return(&models.UserData{ID: userID, Email: "test@example.com", Password: hashedPass}, nil)
				mockStore.EXPECT().ResetFailedLogin(gomock.Any(), "test@example.com").Return(nil)
				mockStore.EXPECT().CreateToken(gomock.Any(), "test@example.com", gomock.Any()).Return(errors.New("token error"))
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			resp, err := svc.SignIn(nil, tt.user)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
				assert.Nil(t, resp)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			}
		})
	}
}

func TestService_RefreshToken(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "test_access_secret")
	t.Setenv("REFRESH_SECRET", "test_refresh_secret")

	svc, mockStore := newTestService(t)

	userID := uuid.NewString()

	// Generate a valid refresh token for tests
	tokenData, err := GenerateToken(userID, email)
	assert.NoError(t, err)

	accessClaim, err := ParseToken(tokenData.AccessToken, "access")
	assert.NoError(t, err)

	tests := []struct {
		name         string
		accessClaim  *models.Claims
		refreshToken string
		mockCall     func()
		wantErr      bool
	}{
		{
			name:         "success",
			accessClaim:  accessClaim,
			refreshToken: tokenData.RefreshToken,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), accessClaim.ClaimUID).Return(false, nil)
				mockStore.EXPECT().DeleteToken(gomock.Any(), accessClaim.Email, accessClaim.ClaimUID, gomock.Any()).Return(nil)
				mockStore.EXPECT().CreateToken(gomock.Any(), accessClaim.Email, gomock.Any()).Return(nil)
			},
		},
		{
			name:         "nil access claim",
			accessClaim:  nil,
			refreshToken: tokenData.RefreshToken,
			mockCall:     func() {},
			wantErr:      true,
		},
		{
			name:         "access token revoked",
			accessClaim:  accessClaim,
			refreshToken: tokenData.RefreshToken,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), accessClaim.ClaimUID).Return(true, nil)
			},
			wantErr: true,
		},
		{
			name:         "IsTokenRevoked error",
			accessClaim:  accessClaim,
			refreshToken: tokenData.RefreshToken,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), accessClaim.ClaimUID).Return(false, errors.New("redis error"))
			},
			wantErr: true,
		},
		{
			name:         "invalid refresh token",
			accessClaim:  accessClaim,
			refreshToken: "invalid-token",
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), accessClaim.ClaimUID).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name:         "delete token error",
			accessClaim:  accessClaim,
			refreshToken: tokenData.RefreshToken,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), accessClaim.ClaimUID).Return(false, nil)
				mockStore.EXPECT().DeleteToken(gomock.Any(), accessClaim.Email, accessClaim.ClaimUID, gomock.Any()).Return(errors.New("del error"))
			},
			wantErr: true,
		},
		{
			name:         "create token error",
			accessClaim:  accessClaim,
			refreshToken: tokenData.RefreshToken,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), accessClaim.ClaimUID).Return(false, nil)
				mockStore.EXPECT().DeleteToken(gomock.Any(), accessClaim.Email, accessClaim.ClaimUID, gomock.Any()).Return(nil)
				mockStore.EXPECT().CreateToken(gomock.Any(), accessClaim.Email, gomock.Any()).Return(errors.New("create error"))
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			resp, err := svc.RefreshToken(nil, tt.accessClaim, tt.refreshToken)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
				assert.Nil(t, resp)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			}
		})
	}
}

func TestService_RevokeToken(t *testing.T) {
	svc, mockStore := newTestService(t)

	claim := &models.Claims{
		Email:    email,
		ClaimUID: uuid.NewString(),
	}

	tests := []struct {
		name     string
		claim    *models.Claims
		mockCall func()
		wantErr  bool
	}{
		{
			name:  "success",
			claim: claim,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), claim.ClaimUID).Return(false, nil)
				mockStore.EXPECT().DeleteToken(gomock.Any(), claim.Email, claim.ClaimUID, "").Return(nil)
			},
		},
		{
			name:     "nil claim",
			claim:    nil,
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name:  "already revoked - idempotent",
			claim: claim,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), claim.ClaimUID).Return(true, nil)
			},
		},
		{
			name:  "IsTokenRevoked error",
			claim: claim,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), claim.ClaimUID).Return(false, errors.New("redis error"))
			},
			wantErr: true,
		},
		{
			name:  "delete token error",
			claim: claim,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), claim.ClaimUID).Return(false, nil)
				mockStore.EXPECT().DeleteToken(gomock.Any(), claim.Email, claim.ClaimUID, "").Return(errors.New("del error"))
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			err := svc.RevokeToken(nil, tt.claim)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
			}
		})
	}
}

func TestService_ValidateTokens(t *testing.T) {
	svc, mockStore := newTestService(t)

	userID := uuid.NewString()
	claim := &models.Claims{
		Email:    email,
		ClaimUID: uuid.NewString(),
	}
	claim.Subject = userID

	tests := []struct {
		name     string
		claim    *models.Claims
		mockCall func()
		wantUID  bool
		wantErr  bool
	}{
		{
			name:  "success",
			claim: claim,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), claim.ClaimUID).Return(false, nil)
			},
			wantUID: true,
		},
		{
			name:     "nil claim",
			claim:    nil,
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name:  "token revoked",
			claim: claim,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), claim.ClaimUID).Return(true, nil)
			},
			wantErr: true,
		},
		{
			name:  "IsTokenRevoked error",
			claim: claim,
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), claim.ClaimUID).Return(false, errors.New("redis error"))
			},
			wantErr: true,
		},
		{
			name: "empty subject",
			claim: &models.Claims{
				Email:    email,
				ClaimUID: uuid.NewString(),
			},
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "invalid uuid subject",
			claim: func() *models.Claims {
				c := &models.Claims{Email: email, ClaimUID: uuid.NewString()}
				c.Subject = "not-a-uuid"
				return c
			}(),
			mockCall: func() {
				mockStore.EXPECT().IsTokenRevoked(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			uid, err := svc.ValidateTokens(nil, tt.claim)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
				assert.Nil(t, uid)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				if tt.wantUID {
					assert.NotNil(t, uid)
				}
			}
		})
	}
}

func TestService_tokenParsing(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "test_access_secret")
	t.Setenv("REFRESH_SECRET", "test_refresh_secret")

	svc := &Service{}

	userID := uuid.NewString()
	tokenData, err := GenerateToken(userID, email)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		tokenType string
		wantErr   bool
	}{
		{
			name:      "valid access token",
			token:     tokenData.AccessToken,
			tokenType: "access",
		},
		{
			name:      "valid refresh token",
			token:     tokenData.RefreshToken,
			tokenType: "refresh",
		},
		{
			name:      "invalid token",
			token:     "invalid",
			tokenType: "access",
			wantErr:   true,
		},
		{
			name:      "invalid token type",
			token:     tokenData.AccessToken,
			tokenType: "unknown",
			wantErr:   true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claim, err := svc.tokenParsing(tt.token, tt.tokenType)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
				assert.Nil(t, claim)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.NotNil(t, claim)
			}
		})
	}
}
