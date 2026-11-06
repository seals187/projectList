package mail

import (
	"fmt"
	"strings"
	"time"
)

// Langs are the supported languages of the e-mail texts.
var Langs = []string{"de", "en", "fr"}

// Lang returns a supported language ("en" otherwise).
func Lang(l string) string {
	for _, x := range Langs {
		if strings.EqualFold(l, x) {
			return x
		}
	}
	return "en"
}

// Invite builds the invitation e-mail.
func Invite(lang, to, name, inviter, link string, expires time.Time) Message {
	d := expires.UTC().Format("2006-01-02 15:04 UTC")
	switch Lang(lang) {
	case "de":
		return Message{To: to, Subject: "Einladung zu ProjectList",
			Body: fmt.Sprintf("Hallo %s\n\n%s hat dich zu ProjectList eingeladen.\n\nÜber diesen Link legst du dein Passwort fest und meldest dich an:\n%s\n\nDer Link ist bis %s gültig und kann nur einmal verwendet werden.\n", name, inviter, link, d)}
	case "fr":
		return Message{To: to, Subject: "Invitation à ProjectList",
			Body: fmt.Sprintf("Bonjour %s\n\n%s vous a invité(e) à ProjectList.\n\nCe lien vous permet de définir votre mot de passe et de vous connecter :\n%s\n\nLe lien est valable jusqu'au %s et ne peut être utilisé qu'une seule fois.\n", name, inviter, link, d)}
	default:
		return Message{To: to, Subject: "Invitation to ProjectList",
			Body: fmt.Sprintf("Hello %s\n\n%s has invited you to ProjectList.\n\nUse this link to set your password and sign in:\n%s\n\nThe link is valid until %s and can only be used once.\n", name, inviter, link, d)}
	}
}

// Reset builds the password reset e-mail.
func Reset(lang, to, name, link string, expires time.Time) Message {
	d := expires.UTC().Format("2006-01-02 15:04 UTC")
	switch Lang(lang) {
	case "de":
		return Message{To: to, Subject: "ProjectList: Passwort zurücksetzen",
			Body: fmt.Sprintf("Hallo %s\n\nFür dein Konto bei ProjectList wurde ein neues Passwort angefordert.\n\nÜber diesen Link legst du ein neues Passwort fest:\n%s\n\nDer Link ist bis %s gültig und kann nur einmal verwendet werden. Danach bist du auf allen Geräten abgemeldet.\n", name, link, d)}
	case "fr":
		return Message{To: to, Subject: "ProjectList : réinitialiser le mot de passe",
			Body: fmt.Sprintf("Bonjour %s\n\nUn nouveau mot de passe a été demandé pour votre compte ProjectList.\n\nCe lien vous permet de définir un nouveau mot de passe :\n%s\n\nLe lien est valable jusqu'au %s et ne peut être utilisé qu'une seule fois. Vous serez ensuite déconnecté(e) sur tous les appareils.\n", name, link, d)}
	default:
		return Message{To: to, Subject: "ProjectList: reset your password",
			Body: fmt.Sprintf("Hello %s\n\nA new password was requested for your ProjectList account.\n\nUse this link to set a new password:\n%s\n\nThe link is valid until %s and can only be used once. Afterwards you are signed out on all devices.\n", name, link, d)}
	}
}
