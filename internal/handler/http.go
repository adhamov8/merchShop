package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"merchShop/internal/handler/mw"
	"merchShop/internal/usecase"
)

type Handler struct {
	service *usecase.Service
}

func NewHandler(service *usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r chi.Router) {
	r.Use(middleware.Logger)

	r.Get("/", h.rootHandler)

	r.Post("/api/auth", h.auth)

	r.Group(func(r chi.Router) {
		r.Use(mw.JWTAuthMiddleware)
		r.Get("/api/info", h.getInfo)
		r.Post("/api/sendCoin", h.sendCoin)
		r.Get("/api/buy/{item}", h.buyMerch)
	})
}

func (h *Handler) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`
<html>
<head>
  <title>Avito Merch Shop</title>
</head>
<body style="font-family: sans-serif;">
  <h1>Добро пожаловать в Merch Shop</h1>
  <p>В этом сервисе вы можете:</p>
  <ul>
    <li>Авторизоваться / зарегистрироваться: <strong>POST /api/auth</strong></li>
    <li>Получить информацию о монетах, инвентаре, истории: <strong>GET /api/info</strong> 
      (требуется Bearer токен в заголовке <code>Authorization</code>)</li>
    <li>Отправить монеты другому пользователю: <strong>POST /api/sendCoin</strong> 
      (также JWT)</li>
    <li>Купить мерч: <strong>GET /api/buy/{item}</strong> (JWT)</li>
  </ul>
  <p>Для закрытых эндпоинтов передавайте заголовок:
    <code>Authorization: Bearer &lt;ваш-токен&gt;</code>
  </p>
</body>
</html>
`))
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (h *Handler) auth(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"errors":"bad request"}`, http.StatusBadRequest)
		return
	}
	user, err := h.service.RegisterOrLogin(r.Context(), req.Username, req.Password)
	if err != nil {
		if err == usecase.ErrInvalidCredentials {
			http.Error(w, `{"errors":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		if err == usecase.ErrWeakPassword {
			http.Error(w, `{"errors":"weak password"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"errors":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	token, err := mw.GenerateJWT(user.ID, user.Username)
	if err != nil {
		http.Error(w, `{"errors":"internal error"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, authResponse{Token: token})
}

func (h *Handler) getInfo(w http.ResponseWriter, r *http.Request) {
	userID := mw.MustGetUserID(r.Context())
	info, err := h.service.GetInfo(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"errors":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	writeJSON(w, info)
}

type sendCoinRequest struct {
	ToUser string `json:"toUser"`
	Amount int    `json:"amount"`
}

func (h *Handler) sendCoin(w http.ResponseWriter, r *http.Request) {
	userID := mw.MustGetUserID(r.Context())

	var req sendCoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"errors":"bad request"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.SendCoin(r.Context(), userID, req.ToUser, req.Amount); err != nil {
		if err == usecase.ErrNotEnoughCoins {
			http.Error(w, `{"errors":"not enough coins"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"errors":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) buyMerch(w http.ResponseWriter, r *http.Request) {
	userID := mw.MustGetUserID(r.Context())
	itemName := chi.URLParam(r, "item")
	if itemName == "" {
		http.Error(w, `{"errors":"item is required"}`, http.StatusBadRequest)
		return
	}
	if err := h.service.BuyMerch(r.Context(), userID, itemName); err != nil {
		if err == usecase.ErrNotEnoughCoins {
			http.Error(w, `{"errors":"not enough coins"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"errors":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}
