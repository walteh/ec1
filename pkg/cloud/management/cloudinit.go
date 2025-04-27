package management

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/kdomanski/iso9660"
	"gopkg.in/yaml.v3"
)

type workingCloudInit struct {
	meta map[string]interface{}
	user map[string]interface{}
}

func addNetworkInitRunCmd(wci *workingCloudInit) {
	// Ensure runcmd section exists to enable and start the service
	runcmd, ok := wci.user["runcmd"].([]interface{})
	if !ok {
		runcmd = []interface{}{}
	}

	dat := `  - |
    # find the NAT gateway (first hop)
    GW=$(ip route | awk '/default/ {print $3}')
    # try up to 10 times
    for i in $(seq 1 10); do
      # report your IP back to host
      curl --retry 3 --retry-delay 2 \
        "http://${GW}:12345/ready?ip=$(hostname -I)"
      sleep 1
    done`

	runcmd = append([]any{dat}, runcmd...)
	wci.user["runcmd"] = runcmd
}

func addAgentBinaryToSystemd(wci *workingCloudInit) {
	// Ensure write_files section exists in user data
	writeFiles, ok := wci.user["write_files"].([]interface{})
	if !ok {
		writeFiles = []interface{}{}
	}

	// Add systemd service unit file
	serviceUnit := map[string]interface{}{
		"path":        "/etc/systemd/system/ec1-agent.service",
		"permissions": "0644",
		"content": `[Unit]
Description=EC1 Agent Service
After=network.target

[Service]
ExecStart=/usr/local/bin/ec1-agent
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`,
	}
	writeFiles = append(writeFiles, serviceUnit)
	wci.user["write_files"] = writeFiles

	// Ensure runcmd section exists to enable and start the service
	runcmd, ok := wci.user["runcmd"].([]interface{})
	if !ok {
		runcmd = []interface{}{}
	}

	// Add commands to enable and start the service
	runcmd = append(runcmd,
		"systemctl daemon-reload",
		"systemctl enable ec1-agent.service",
		"systemctl start ec1-agent.service")

	wci.user["runcmd"] = runcmd
}

func CreateCloudInitISO(ctx context.Context, meta, user string, extraFiles map[string]string, manipulations ...func(*workingCloudInit)) (string, func(), error) {
	// create a temp dir

	writer, err := iso9660.NewWriter()
	if err != nil {
		return "", nil, err
	}
	defer writer.Cleanup()
	var parsedMeta map[string]interface{}
	if err := yaml.Unmarshal([]byte(meta), &parsedMeta); err != nil {
		return "", nil, err
	}

	var parsedUser map[string]interface{}
	if err := yaml.Unmarshal([]byte(user), &parsedUser); err != nil {
		return "", nil, err
	}

	if parsedMeta == nil {
		parsedMeta = map[string]interface{}{}
	}

	if parsedUser == nil {
		parsedUser = map[string]interface{}{}
	}

	wci := workingCloudInit{
		meta: parsedMeta,
		user: parsedUser,
	}

	/// apply changes to the parsed meta and user data
	for _, manipulation := range manipulations {
		manipulation(&wci)
	}

	backMeta, err := yaml.Marshal(wci.meta)
	if err != nil {
		return "", nil, err
	}

	backUser, err := yaml.Marshal(wci.user)
	if err != nil {
		return "", nil, err
	}

	for f, content := range map[string]string{"meta-data": string(backMeta), "user-data": string(backUser)} {
		if err = writer.AddFile(strings.NewReader(content), f); err != nil {
			return "", nil, err
		}
	}

	for f, content := range extraFiles {
		if err = writer.AddFile(strings.NewReader(content), filepath.Join("extra", f)); err != nil {
			return "", nil, err
		}
	}

	fle, err := os.CreateTemp("", "cloud-init.iso")
	if err != nil {
		return "", nil, err
	}
	defer fle.Close()

	if err := writer.WriteTo(fle, "cidata"); err != nil {
		return "", nil, err
	}

	return fle.Name(), func() { os.Remove(fle.Name()) }, nil
}
