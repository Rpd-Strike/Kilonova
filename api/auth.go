package api

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

/*
	NOTE: Session expires after 30 days
	Cookie should look like this:
	cookie := &http.Cookie{
		Name:     "kn-sessionid",
		Value:    sid,
		Path:     "/",
		HttpOnly: false,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	}
*/

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Language string `json:"language"`
		Theme    string `json:"theme"`

		CaptchaID       string `json:"captcha_id"`
		CaptchaResponse string `json:"captcha_response"`
	}

	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	ip, ua := s.base.GetRequestInfo(r)
	if s.base.MustSolveCaptcha(r.Context(), ip) {
		if auth.CaptchaID == "" || auth.CaptchaResponse == "" {
			errorData(w, struct {
				ID  string `json:"captcha_id"`
				Key string `json:"translation_key"`
			}{
				ID:  s.base.NewCaptchaID(),
				Key: "auth.captcha.must_solve",
			}, http.StatusPreconditionRequired)
			return
		}
		if !s.base.CheckCaptcha(auth.CaptchaID, auth.CaptchaResponse) {
			errorData(w, struct {
				ID  string `json:"captcha_id"`
				Key string `json:"translation_key"`
			}{
				ID:  s.base.NewCaptchaID(),
				Key: "auth.captcha.invalid",
			}, http.StatusBadRequest)
			return
		}
	}

	uid, status := s.base.Signup(r.Context(), auth.Email, auth.Username, auth.Password, auth.Language, kilonova.PreferredTheme(auth.Theme))
	if status != nil {
		errorData(w, struct {
			ID   string `json:"captcha_id"`
			Text string `json:"text"`
		}{
			ID:   s.base.NewCaptchaID(),
			Text: status.Text,
		}, status.Code)
		return
	}

	go func() {
		if err := s.base.LogSignup(context.Background(), uid, ip, &ua); err != nil {
			zap.S().Warn(err)
		}
	}()

	sid, err1 := s.base.CreateSession(r.Context(), uid)
	if err1 != nil {
		err1.WriteError(w)
		return
	}
	returnData(w, sid)
}

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth struct {
		Username string
		Password string
	}

	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	user, status := s.base.Login(r.Context(), auth.Username, auth.Password)
	if status != nil {
		status.WriteError(w)
		return
	}

	if user.LockedLogin && !user.Admin {
		// Lockout but don't lockout admins
		errorData(w, "Login for this account has been restricted by an administrator", 401)
		return
	}

	sid, err1 := s.base.CreateSession(r.Context(), user.ID)
	if err1 != nil {
		err1.WriteError(w)
		return
	}
	returnData(w, sid)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	s.base.RemoveSession(r.Context(), getAuthHeader(r))
	returnData(w, "Logged out")
}

func (s *API) extendSession(w http.ResponseWriter, r *http.Request) {
	h := getAuthHeader(r)
	if h == "" {
		zap.S().Warn("Empty session on endpoint that must be authed")
		returnData(w, nil)
		return
	}
	exp, err := s.base.ExtendSession(r.Context(), h)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, exp)
}
