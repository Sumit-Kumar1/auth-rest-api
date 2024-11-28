package handler

import (
	"auth-rest-api/internal/models"
	"auth-rest-api/internal/server"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandler_SignUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockServicer(ctrl)
	h := New(mockService)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx := context.WithValue(context.Background(), server.Logger, logger)

	tests := []struct {
		name        string
		requestBody json.RawMessage
		mockCall    func()
		expCode     int
	}{
		{
			name:        "Successful SignUp",
			requestBody: json.RawMessage(`{"email":"testuser@gmail.com","password":"12345"}`),
			mockCall: func() {
				mockService.EXPECT().SignUp(ctx, gomock.Any()).Return(nil)
			},
			expCode: http.StatusCreated,
		},
		{
			name:     "Request Body Missing",
			mockCall: func() {},
			expCode:  http.StatusBadRequest,
		},
		{
			name:        "Invalid JSON",
			requestBody: json.RawMessage(`{"email":"testuser@gmail.com""password":"12345"}`),
			mockCall:    func() {},
			expCode:     http.StatusBadRequest,
		},
		{
			name:        "User  Already Exists",
			requestBody: json.RawMessage(`{"email":"testuser@gmail.com","password":"12345"}`),
			mockCall: func() {
				mockService.EXPECT().SignUp(ctx, gomock.Any()).Return(models.ErrUserAlreadyExists)
			},
			expCode: http.StatusConflict,
		},
		{
			name:        "Internal Server Error",
			requestBody: json.RawMessage(`{"email":"testuser@gmail.com","password":"12345"}`),
			mockCall: func() {
				mockService.EXPECT().SignUp(ctx, gomock.Any()).Return(models.ErrDBNotConnected)
			},
			expCode: http.StatusInternalServerError,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(ctx, "POST", "/sign-up", bytes.NewBuffer(tt.requestBody))

			h.SignUp(w, r)

			assert.Equalf(t, tt.expCode, w.Code, "TEST[%d] Failed - %s", i, tt.name)
		})
	}
}
