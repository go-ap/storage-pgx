package cmd

import (
	"fmt"
	"github.com/go-ap/activitypub"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"net/url"
	"time"
)

type Control struct {
	Conf        config.Options
	Host        string
	BaseURL     *url.URL
	AuthStorage osin.Storage
	Storage     storage.Repository
}

type PasswordChanger interface {
	PasswordSet(as.Item, []byte) error
}

func (c *Control) AddActor(preferredUsername string, typ as.ActivityVocabularyType, pw []byte) (*auth.Person, error) {
	self := ap.Self(as.IRI(c.BaseURL.String()))
	now := time.Now()
	p := auth.Person{
		Person: activitypub.Person{
			Parent: as.Parent{
				Type: typ,
				// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
				AttributedTo: self.GetLink(),
				Audience:     as.ItemCollection{auth.ActivityStreamsPublicNS},
				Generator:    self.GetLink(),
				Published:    now,
				Summary: as.NaturalLanguageValues{
					{as.NilLangRef, "Generated actor"},
				},
				Updated: now,
			},
			PreferredUsername: as.NaturalLanguageValues{
				{as.NilLangRef, preferredUsername},
			},
		},
	}

	// TODO(marius): add annotations for the errors
	if gen, ok := c.Storage.(storage.IDGenerator); ok {
		id, err := gen.GenerateID(p, self)
		if err != nil {
			return nil, err
		}
		p.ID = id
		p.URL = p.GetLink()
		p.Inbox = as.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Inbox))
		p.Outbox = as.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Outbox))
		p.Liked = as.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Liked))
		p.Likes = as.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Likes))

		p.Endpoints = &activitypub.Endpoints{
			SharedInbox:                self.Inbox.GetLink(),
			OauthAuthorizationEndpoint: as.IRI(fmt.Sprintf("%s/oauth/authorize", self.URL)),
			OauthTokenEndpoint:         as.IRI(fmt.Sprintf("%s/oauth/token", self.URL)),
		}
	}
	it, err := c.Storage.SaveActor(p)
	if err != nil {
		return nil, err
	}

	saved, err := auth.ToPerson(it)
	if err != nil {
		return nil, err
	}

	if pw != nil {
		// TODO(marius): add the password somewhere in the actor's data
		if pwManager, ok := c.Storage.(PasswordChanger); ok {
			err := pwManager.PasswordSet(saved.GetLink(), pw)
			if err != nil {
				return saved, err
			}
		}
	}

	return saved, nil
}

func (c *Control) DeleteActor(id string) error {
	self := ap.Self(as.IRI(c.BaseURL.String()))
	var iri as.IRI
	if u, err := url.Parse(id); err != nil {
		iri = as.IRI(fmt.Sprintf("%s/%s/%s", self.ID, ap.ActorsType, id))
	} else {
		iri = as.IRI(u.String())
	}
	it, cnt, err := c.Storage.LoadActors(iri)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return errors.Newf("")
	}
	_, err = c.Storage.DeleteActor(it.First())
	return err
}