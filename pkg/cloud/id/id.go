package id

import (
	"fmt"
	"strings"

	"github.com/rs/xid"
)

type ID struct {
	xid   xid.ID
	group string
}

func NewID(name string) ID {
	return ID{
		xid:   xid.New(),
		group: name,
	}
}

func (id ID) String() string {
	return fmt.Sprintf("%s-%s", id.group, id.xid.String())
}

func ParseID(id string) (ID, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return ID{}, fmt.Errorf("invalid ID: %s", id)
	}

	xid, err := xid.FromString(parts[1])
	if err != nil {
		return ID{}, fmt.Errorf("invalid ID: %s", id)
	}

	return ID{group: parts[0], xid: xid}, nil
}
