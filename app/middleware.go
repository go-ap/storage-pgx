package app

import (
	"context"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/validation"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
)

// Repo adds an implementation of the storage.Loader to a Request's context so it can be used
// further in the middleware chain
func Repo(loader storage.Loader) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			newCtx := context.WithValue(ctx, handlers.RepositoryKey, loader)
			next.ServeHTTP(w, r.WithContext(newCtx))
		}
		return http.HandlerFunc(fn)
	}
}

// Validator adds an implementation of the validation.ActivityValidator to a Request's context so it can be used
// further in the middleware chain
func Validator(v validation.ActivityValidator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			newCtx := context.WithValue(ctx, validation.ValidatorKey, v)
			next.ServeHTTP(w, r.WithContext(newCtx))
		}
		return http.HandlerFunc(fn)
	}
}

// ActorFromAuthHeader tries to load a local actor from the OAuth2 or HTTP Signatures Authorization headers
func ActorFromAuthHeader(os *osin.Server, st storage.ActorLoader, l logrus.FieldLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// TODO(marius): move this to the auth package and also add the possibility of getting the logger as a parameter
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := auth.New(reqURL(r), os, st, l)
			act, err := s.LoadActorFromAuthHeader(r)
			if err != nil {
				if errors.IsUnauthorized(err) {
					if challenge := errors.Challenge(err); len(challenge) > 0 {
						w.Header().Add("WWW-Authenticate", challenge)
					}
				}
				l.Warnf("%s", err)
			}
			if act != nil {
				r = r.WithContext(context.WithValue(r.Context(), auth.ActorKey, act))
			}
			next.ServeHTTP(w, r)
		})
	}
}
