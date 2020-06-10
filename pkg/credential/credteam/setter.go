package credteam

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ZupIT/ritchie-cli/pkg/credential"
	"github.com/ZupIT/ritchie-cli/pkg/http/headers"
	"github.com/ZupIT/ritchie-cli/pkg/rcontext"
	"github.com/ZupIT/ritchie-cli/pkg/server"
	"github.com/ZupIT/ritchie-cli/pkg/session"
)

const urlCreatePattern = "%s/credentials/%s"

type Setter struct {
	serverFinder   server.Finder
	httpClient     *http.Client
	sessionManager session.Manager
	ctxFinder      rcontext.Finder
}

func NewSetter(serverFinder server.Finder, hc *http.Client, sm session.Manager, cf rcontext.Finder) Setter {
	return Setter{
		serverFinder:   serverFinder,
		httpClient:     hc,
		sessionManager: sm,
		ctxFinder:      cf,
	}
}

func (s Setter) Set(cred credential.Detail) error {
	sess, err := s.sessionManager.Current()
	if err != nil {
		return err
	}

	ctx, err := s.ctxFinder.Find()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&cred)
	if err != nil {
		return err
	}

	cfg, err := s.serverFinder.Find()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(urlCreatePattern, cfg.URL, cred.Type)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headers.XOrg, sess.Organization)
	req.Header.Set(headers.XCtx, ctx.Current)
	req.Header.Set(headers.Authorization, sess.AccessToken)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	default:
		log.Printf("Status code: %v", resp.StatusCode)
		return errors.New(string(b))
	}
}
