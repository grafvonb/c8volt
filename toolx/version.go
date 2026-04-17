package toolx

import (
	"errors"
	"fmt"
	"strings"
)

var ErrUnknownCamundaVersion = errors.New("unknown Camunda version")

const (
	V87 CamundaVersion = "8.7"
	V88 CamundaVersion = "8.8"
	V89 CamundaVersion = "8.9"

	CurrentCamundaVersion = V88
)

type CamundaVersion string

func (v CamundaVersion) String() string {
	switch v {
	case V87:
		return "8.7"
	case V88:
		return "8.8"
	case V89:
		return "8.9"
	default:
		return "unknown"
	}
}

func (v CamundaVersion) FilePrefix() string {
	switch v {
	case V87:
		return "C87_"
	case V88:
		return "C88_"
	case V89:
		return "C89_"
	default:
		return "unknown"
	}
}

func NormalizeCamundaVersion(s string) (CamundaVersion, error) {
	v := strings.TrimSpace(strings.ToLower(s))
	switch v {
	case "8.7", "87", "v87", "v8.7":
		return V87, nil
	case "8.8", "88", "v88", "v8.8":
		return V88, nil
	case "8.9", "89", "v89", "v8.9":
		return V89, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownCamundaVersion, v)
	}
}

func SupportedCamundaVersions() []CamundaVersion {
	return []CamundaVersion{V87, V88, V89}
}

func ImplementedCamundaVersions() []CamundaVersion {
	return []CamundaVersion{V87, V88}
}

func SupportedCamundaVersionsString() string {
	return joinCamundaVersions(SupportedCamundaVersions())
}

func ImplementedCamundaVersionsString() string {
	return joinCamundaVersions(ImplementedCamundaVersions())
}

func joinCamundaVersions(versions []CamundaVersion) string {
	var parts []string
	for _, v := range versions {
		parts = append(parts, v.String())
	}
	return strings.Join(parts, ", ")
}
