package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"auth-rest-api/internal/models"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler_SignUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockServicer(ctrl)
	h := New(mockService)

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
				mockService.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
			},
			expCode: http.StatusCreated,
		},
		{
			name: "Request Body Missing",
			mockCall: func() {
				mockService.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(models.ErrBadRequest(models.ErrEmailRequired))
			},
			expCode: http.StatusBadRequest,
		},
		{
			name:        "Invalid JSON",
			requestBody: json.RawMessage(`{"email":"testuser@gmail.com""password":"12345"}`),
			mockCall:    func() {},
			expCode:     http.StatusBadRequest,
		},
		{
			name:        "User Already Exists",
			requestBody: json.RawMessage(`{"email":"testuser@gmail.com","password":"12345"}`),
			mockCall: func() {
				mockService.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(models.ErrUserAlreadyExists)
			},
			expCode: http.StatusConflict,
		},
		{
			name:        "Internal Server Error",
			requestBody: json.RawMessage(`{"email":"testuser@gmail.com","password":"12345"}`),
			mockCall: func() {
				mockService.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(models.ErrDBNotConnected)
			},
			expCode: http.StatusInternalServerError,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockCall()

			e := echo.New()
			var req *http.Request
			if tt.requestBody != nil {
				req = httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(tt.requestBody))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			} else {
				req = httptest.NewRequest(http.MethodPost, "/signup", nil)
			}

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = h.SignUp(c)

			assert.Equalf(t, tt.expCode, rec.Code, "TEST[%d] Failed - %s", i, tt.name)
		})
	}
}
