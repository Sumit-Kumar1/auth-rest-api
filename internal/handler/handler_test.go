package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"auth-rest-api/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/container"
	gofrHTTP "gofr.dev/pkg/gofr/http"
)

const testFailStr = "TEST[%d] Failed - %s"

func newTestHandler(t *testing.T) (*Handler, *MockServicer, *container.Container) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockService := NewMockServicer(ctrl)
	mockContainer, _ := container.NewMockContainer(t)

	return New(mockService), mockService, mockContainer
}

func newGofrContext(t *testing.T, method, path string, body []byte, mockContainer *container.Container, claims *models.Claims) *gofr.Context {
	t.Helper()

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, http.NoBody)
	}

	ctx := req.Context()
	if claims != nil {
		ctx = context.WithValue(ctx, models.CtxClaimKey, claims)
	}

	req = req.WithContext(ctx)

	gofrReq := gofrHTTP.NewRequest(req)

	return &gofr.Context{
		Context:   ctx,
		Request:   gofrReq,
		Container: mockContainer,
	}
}

func TestHandler_SignUp(t *testing.T) {
	h, mockService, mockContainer := newTestHandler(t)

	tests := []struct {
		name     string
		body     []byte
		mockCall func()
		wantErr  bool
	}{
		{
			name: "success",
			body: []byte(`{"email":"test@example.com","password":"Test@1234"}`),
			mockCall: func() {
				mockService.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "service error",
			body: []byte(`{"email":"test@example.com","password":"Test@1234"}`),
			mockCall: func() {
				mockService.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(errors.New("signup error"))
			},
			wantErr: true,
		},
		{
			name:     "invalid json",
			body:     []byte(`{"email":"test@example.com""password":"Test@1234"}`),
			mockCall: func() {},
			wantErr:  true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			c := newGofrContext(t, http.MethodPost, "/signup", tt.body, mockContainer, nil)

			resp, err := h.SignUp(c)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.Equal(t, "user created successfully", resp)
			}
		})
	}
}

func TestHandler_SignIn(t *testing.T) {
	h, mockService, mockContainer := newTestHandler(t)

	tests := []struct {
		name     string
		body     []byte
		mockCall func()
		wantErr  bool
	}{
		{
			name: "success",
			body: []byte(`{"email":"test@example.com","password":"Test@1234"}`),
			mockCall: func() {
				mockService.EXPECT().SignIn(gomock.Any(), gomock.Any()).
					Return(&models.TokenResponse{AccessToken: "acc-token", RefreshToken: "ref-token"}, nil)
			},
		},
		{
			name: "service error",
			body: []byte(`{"email":"test@example.com","password":"Test@1234"}`),
			mockCall: func() {
				mockService.EXPECT().SignIn(gomock.Any(), gomock.Any()).
					Return(nil, models.ErrPasswordMismatch{})
			},
			wantErr: true,
		},
		{
			name:     "invalid json",
			body:     []byte(`invalid`),
			mockCall: func() {},
			wantErr:  true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			c := newGofrContext(t, http.MethodPost, "/signin", tt.body, mockContainer, nil)

			resp, err := h.SignIn(c)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)

				userResp, ok := resp.(models.UserResp)
				assert.True(t, ok)
				assert.NotEmpty(t, userResp.AccessToken)
				assert.NotEmpty(t, userResp.RefreshToken)
			}
		})
	}
}

func TestHandler_RefreshToken(t *testing.T) {
	h, mockService, mockContainer := newTestHandler(t)

	claims := &models.Claims{
		Email:    "test@example.com",
		ClaimUID: uuid.NewString(),
	}

	tests := []struct {
		name     string
		body     []byte
		claims   *models.Claims
		mockCall func()
		wantErr  bool
	}{
		{
			name:   "success",
			body:   []byte(`{"refreshToken":"valid-refresh-token"}`),
			claims: claims,
			mockCall: func() {
				mockService.EXPECT().RefreshToken(gomock.Any(), gomock.Any(), "valid-refresh-token").
					Return(&models.TokenResponse{AccessToken: "new-acc", RefreshToken: "new-ref"}, nil)
			},
		},
		{
			name:     "no claims in context",
			body:     []byte(`{"refreshToken":"token"}`),
			claims:   nil,
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name:   "service error - token revoked",
			body:   []byte(`{"refreshToken":"token"}`),
			claims: claims,
			mockCall: func() {
				mockService.EXPECT().RefreshToken(gomock.Any(), gomock.Any(), "token").
					Return(nil, models.ErrTokenRevoked{})
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			c := newGofrContext(t, http.MethodPost, "/refresh", tt.body, mockContainer, tt.claims)

			resp, err := h.RefreshToken(c)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)

				userResp, ok := resp.(models.UserResp)
				assert.True(t, ok)
				assert.NotEmpty(t, userResp.AccessToken)
				assert.NotEmpty(t, userResp.RefreshToken)
			}
		})
	}
}

func TestHandler_RevokeToken(t *testing.T) {
	h, mockService, mockContainer := newTestHandler(t)

	claims := &models.Claims{
		Email:    "test@example.com",
		ClaimUID: uuid.NewString(),
	}

	tests := []struct {
		name     string
		claims   *models.Claims
		mockCall func()
		wantErr  bool
	}{
		{
			name:   "success",
			claims: claims,
			mockCall: func() {
				mockService.EXPECT().RevokeToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:     "no claims in context",
			claims:   nil,
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name:   "service error",
			claims: claims,
			mockCall: func() {
				mockService.EXPECT().RevokeToken(gomock.Any(), gomock.Any()).Return(errors.New("revoke error"))
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			c := newGofrContext(t, http.MethodPost, "/revoke", nil, mockContainer, tt.claims)

			resp, err := h.RevokeToken(c)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.Equal(t, "token revoked successfully", resp)
			}
		})
	}
}

func TestHandler_Validate(t *testing.T) {
	h, mockService, mockContainer := newTestHandler(t)

	userID := uuid.New()

	claims := &models.Claims{
		Email:    "test@example.com",
		ClaimUID: uuid.NewString(),
	}

	tests := []struct {
		name     string
		claims   *models.Claims
		mockCall func()
		wantErr  bool
	}{
		{
			name:   "success",
			claims: claims,
			mockCall: func() {
				mockService.EXPECT().ValidateTokens(gomock.Any(), gomock.Any()).Return(&userID, nil)
			},
		},
		{
			name:     "no claims in context",
			claims:   nil,
			mockCall: func() {},
			wantErr:  true,
		},
		{
			name:   "service error",
			claims: claims,
			mockCall: func() {
				mockService.EXPECT().ValidateTokens(gomock.Any(), gomock.Any()).Return(nil, models.ErrTokenRevoked{})
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			c := newGofrContext(t, http.MethodPost, "/validate", nil, mockContainer, tt.claims)

			resp, err := h.Validate(c)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestExtractClaimFromCtx(t *testing.T) {
	_, _, mockContainer := newTestHandler(t)

	tests := []struct {
		name    string
		claims  *models.Claims
		wantErr bool
	}{
		{
			name:   "success",
			claims: &models.Claims{Email: "test@example.com", ClaimUID: uuid.NewString()},
		},
		{
			name:    "no claims",
			claims:  nil,
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newGofrContext(t, http.MethodPost, "/validate", nil, mockContainer, tt.claims)

			claims, err := extractClaimFromCtx(c)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
				assert.Nil(t, claims)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.NotNil(t, claims)
				assert.Equal(t, tt.claims.Email, claims.Email)
			}
		})
	}
}
