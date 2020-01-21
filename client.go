package webdav

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/emersion/go-webdav/internal"
)

type Client struct {
	ic *internal.Client
}

func NewClient(c *http.Client, endpoint string) (*Client, error) {
	ic, err := internal.NewClient(c, endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{ic}, nil
}

func (c *Client) SetBasicAuth(username, password string) {
	c.ic.SetBasicAuth(username, password)
}

func (c *Client) FindCurrentUserPrincipal() (string, error) {
	propfind := internal.NewPropNamePropfind(internal.CurrentUserPrincipalName)

	resp, err := c.ic.PropfindFlat("/", propfind)
	if err != nil {
		return "", err
	}

	var prop internal.CurrentUserPrincipal
	if err := resp.DecodeProp(&prop); err != nil {
		return "", err
	}
	if prop.Unauthenticated != nil {
		return "", fmt.Errorf("webdav: unauthenticated")
	}

	return prop.Href, nil
}

type fileInfo struct {
	filename string
	size     int64
	modTime  time.Time
	isDir    bool
}

func (fi *fileInfo) Name() string {
	return fi.filename
}

func (fi *fileInfo) Size() int64 {
	return fi.size
}

func (fi *fileInfo) Mode() os.FileMode {
	if fi.isDir {
		return os.ModePerm | os.ModeDir
	} else {
		return os.ModePerm
	}
}

func (fi *fileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *fileInfo) IsDir() bool {
	return fi.isDir
}

func (fi *fileInfo) Sys() interface{} {
	return nil
}

func (c *Client) Stat(name string) (os.FileInfo, error) {
	// TODO: getetag, getcontenttype
	propfind := internal.NewPropNamePropfind(
		internal.ResourceTypeName,
		internal.GetContentLengthName,
		internal.GetLastModifiedName,
	)
	resp, err := c.ic.PropfindFlat(name, propfind)
	if err != nil {
		return nil, err
	}

	filename, _ := path.Split(name)
	fi := &fileInfo{filename: filename}

	var resType internal.ResourceType
	if err := resp.DecodeProp(&resType); err != nil {
		return nil, err
	}
	if resType.Is(internal.CollectionName) {
		fi.isDir = true
	} else {
		var getLen internal.GetContentLength
		var getMod internal.GetLastModified
		if err := resp.DecodeProp(&getLen, &getMod); err != nil {
			return nil, err
		}

		fi.size = getLen.Length
		fi.modTime = time.Time(getMod.LastModified)
	}

	return fi, nil
}

func (c *Client) Open(name string) (io.ReadCloser, error) {
	req, err := c.ic.NewRequest(http.MethodGet, name, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.ic.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}
