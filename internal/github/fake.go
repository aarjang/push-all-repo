package github

import (
	"context"
	"time"
)

// CreateCall records one CreateRepo invocation on the fake.
type CreateCall struct {
	Path    string
	Name    string
	Private bool
}

// FakeClient is a scripted Client for tests.
type FakeClient struct {
	Installed bool
	Logged    bool
	// URL returned on success; when empty "https://github.com/me/<name>" is used.
	URL string
	// Errs are returned in order, one per CreateRepo call; nil means success.
	Errs  []error
	Calls []CreateCall
}

// Available implements Client.
func (f *FakeClient) Available() bool { return f.Installed }

// LoggedIn implements Client.
func (f *FakeClient) LoggedIn(context.Context) bool { return f.Installed && f.Logged }

// CreateRepo implements Client.
func (f *FakeClient) CreateRepo(_ context.Context, path, name string, private bool, _ time.Duration) (string, error) {
	f.Calls = append(f.Calls, CreateCall{Path: path, Name: name, Private: private})
	if len(f.Errs) > 0 {
		err := f.Errs[0]
		f.Errs = f.Errs[1:]
		if err != nil {
			return "", err
		}
	}
	if f.URL != "" {
		return f.URL, nil
	}
	return "https://github.com/me/" + name, nil
}
