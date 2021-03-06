package controller

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"

	"github.com/chanceeakin/app-for-emma/server/app/model"
	"github.com/chanceeakin/app-for-emma/server/app/shared/passhash"
	"github.com/chanceeakin/app-for-emma/server/app/shared/response"
	"github.com/chanceeakin/app-for-emma/server/app/shared/session"
	"github.com/chanceeakin/app-for-emma/server/app/shared/view"

	"github.com/gorilla/sessions"
	"github.com/josephspurrier/csrfbanana"
)

const (
	// Name of the session variable that tracks login attempts
	sessLoginAttempt = "login_attempt"
)

// loginAttempt increments the number of login attempts in sessions variable
func loginAttempt(sess *sessions.Session) {
	// Log the attempt
	if sess.Values[sessLoginAttempt] == nil {
		sess.Values[sessLoginAttempt] = 1
	} else {
		sess.Values[sessLoginAttempt] = sess.Values[sessLoginAttempt].(int) + 1
	}
}

// LoginGET displays the login page
func LoginGET(w http.ResponseWriter, r *http.Request) {
	// Get session
	sess := session.Instance(r)

	// Display the view
	v := view.New(r)
	v.Name = "login/login"
	v.Vars["token"] = csrfbanana.Token(w, r, sess)
	// Refill any form fields
	view.Repopulate([]string{"email"}, r.Form, v.Vars)
	v.Render(w)
}

// LoginPOST handles the login form submission
func LoginPOST(w http.ResponseWriter, r *http.Request) {
	// Get session
	sess := session.Instance(r)

	// Prevent brute force login attempts by not hitting MySQL and pretending like it was invalid :-)
	if sess.Values[sessLoginAttempt] != nil && sess.Values[sessLoginAttempt].(int) >= 5 {
		log.Println("Brute force login prevented")
		sess.AddFlash(view.Flash{Message: "Sorry, no brute force :-)", Class: view.FlashNotice})
		sess.Save(r, w)
		LoginGET(w, r)
		return
	}

	// Validate with required fields
	if validate, missingField := view.Validate(r, []string{"email", "password"}); !validate {
		sess.AddFlash(view.Flash{Message: "Field missing: " + missingField, Class: view.FlashError})
		sess.Save(r, w)
		LoginGET(w, r)
		return
	}

	// Form values
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Get database result
	result, err := model.UserByEmail(email)

	// Determine if user exists
	if err == model.ErrNoResult {
		loginAttempt(sess)
		sess.AddFlash(view.Flash{Message: "Password is incorrect - Attempt: " + fmt.Sprintf("%v", sess.Values[sessLoginAttempt]), Class: view.FlashWarning})
		sess.Save(r, w)
	} else if err != nil {
		// Display error message
		log.Println(err)
		sess.AddFlash(view.Flash{Message: "There was an error. Please try again later.", Class: view.FlashError})
		sess.Save(r, w)
	} else if passhash.MatchString(result.Password, password) {
		if result.StatusID != 1 {
			// User inactive and display inactive message
			sess.AddFlash(view.Flash{Message: "Account is inactive so login is disabled.", Class: view.FlashNotice})
			sess.Save(r, w)
		} else {
			// Login successfully
			session.Empty(sess)
			sess.AddFlash(view.Flash{Message: "Login successful!", Class: view.FlashSuccess})
			sess.Values["id"] = result.UserID()
			sess.Values["email"] = email
			sess.Values["first_name"] = result.FirstName
			sess.Values["role"] = result.Role
			sess.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	} else {
		loginAttempt(sess)
		sess.AddFlash(view.Flash{Message: "Password is incorrect - Attempt: " + fmt.Sprintf("%v", sess.Values[sessLoginAttempt]), Class: view.FlashWarning})
		sess.Save(r, w)
	}

	// Show the login page again
	LoginGET(w, r)
}

// LoginInput is the expected data shape for a login attempt from an iPhone.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// IphoneLoginPOST handles the login form submission
func IphoneLoginPOST(w http.ResponseWriter, r *http.Request) {
	// Get session
	sess := session.Instance(r)

	// Prevent brute force login attempts by not hitting MySQL and pretending like it was invalid :-)
	if sess.Values[sessLoginAttempt] != nil && sess.Values[sessLoginAttempt].(int) >= 5 {
		log.Println("Brute force login prevented")
		response.SendError(w, http.StatusForbidden, "Brute Force register prevented.")
		return
	}

	var l LoginInput

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&l); err != nil {
		log.Println(err)
		response.SendError(w, http.StatusBadRequest, "An error occured")
		return
	}
	email := l.Email
	password := l.Password

	// Get database result
	result, err := model.UserByEmail(email)

	// Determine if user exists
	if err == model.ErrNoResult {
		loginAttempt(sess)
		response.SendError(w, http.StatusNotFound, "No user found")
	} else if err != nil {
		// Display error message
		log.Println(err)
		response.SendError(w, http.StatusInternalServerError, "An error occurred")
	} else if passhash.MatchString(result.Password, password) {
		if result.StatusID != 1 {
			// User inactive and display inactive message
			response.Send(w, http.StatusForbidden, "User inactive", 0, nil)
		} else {
			// Login successfully
			// DECLARE A RETURN VALUE
			session.Empty(sess)
			sess.Values["id"] = result.UserID()
			sess.Values["email"] = email
			sess.Values["first_name"] = result.FirstName
			sess.Values["role"] = result.Role
			sess.Save(r, w)

			values := map[string]interface{}{"id": result.UserID(), "email": email, "firstName": result.FirstName, "lastName": result.LastName, "role": result.Role}
			response.SendJSON(w, values)
			return
		}
	} else {
		loginAttempt(sess)
		response.SendError(w, http.StatusForbidden, "Incorrect Password")
	}
	return
}

// LogoutGET clears the session and logs the user out
func LogoutGET(w http.ResponseWriter, r *http.Request) {
	// Get session
	sess := session.Instance(r)

	// If user is authenticated
	if sess.Values["id"] != nil {
		session.Empty(sess)
		sess.AddFlash(view.Flash{Message: "Goodbye!", Class: view.FlashNotice})
		sess.Save(r, w)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// IphoneLogoutGET clears the session and logs the user out
func IphoneLogoutGET(w http.ResponseWriter, r *http.Request) {
	// Get session
	sess := session.Instance(r)
	// If user is authenticated
	if sess.Values["id"] != nil {
		session.Empty(sess)
		sess.Save(r, w)
	}

	values := map[string]interface{}{"message": "User logged out", "success": true}
	response.SendJSON(w, values)
	return
}
