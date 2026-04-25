package common

import (
	"bytes"
	"mime/multipart"
	"net/textproto"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// BuildDeploymentBody creates the multipart request body used by Camunda deployment endpoints.
// tenantID is written as the optional "tenantId" form field, and each deployment unit is written
// as a "resources" file part using the unit name as the submitted filename. The returned
// content type includes the generated multipart boundary and must be passed with the body.
func BuildDeploymentBody(tenantID string, units []d.DeploymentUnitData) (string, *bytes.Reader, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if tenantID != "" {
		if err := w.WriteField("tenantId", tenantID); err != nil {
			return "", nil, err
		}
	}
	for _, u := range units {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="resources"; filename="`+u.Name+`"`)
		part, err := w.CreatePart(h)
		if err != nil {
			return "", nil, err
		}
		if _, err = part.Write(u.Data); err != nil {
			return "", nil, err
		}
	}
	if err := w.Close(); err != nil {
		return "", nil, err
	}
	return w.FormDataContentType(), bytes.NewReader(buf.Bytes()), nil
}
