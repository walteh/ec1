FROM ubuntu:22.04

# Set up environment
ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"
ENV GOPROXY=https://proxy.golang.org
ENV GOSUMDB=sum.golang.org
ENV GOPRIVATE=github.com/walteh

# Install system dependencies
RUN apt-get update && apt-get install -y \
	build-essential \
	git \
	curl \
	wget \
	jq \
	tmux \
	htop \
	watch \
	tree \
	unzip \
	ca-certificates \
	gnupg \
	lsb-release \
	&& rm -rf /var/lib/apt/lists/*

# Install Go 1.24.3
RUN cd /tmp \
	&& wget -q https://go.dev/dl/go1.24.3.linux-amd64.tar.gz \
	&& rm -rf /usr/local/go \
	&& tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz \
	&& rm go1.24.3.linux-amd64.tar.gz

# Install Go tools
RUN go install gotest.tools/gotestsum@latest \
	&& go install github.com/walteh/retab/v2/cmd/retab@latest \
	&& go install golang.org/x/tools/cmd/goimports@latest \
	&& go install github.com/go-delve/delve/cmd/dlv@latest

# Set working directory
WORKDIR /workspace

# Copy and build gow tool when container starts
COPY . .
RUN cd cmd/gow && go build -o ../../gow . && cd /workspace && chmod +x ./gow

# Set up aliases in bashrc
RUN echo 'alias gow="./gow"' >> ~/.bashrc \
	&& echo 'alias gowtest="./gow test -function-coverage -v"' >> ~/.bashrc \
	&& echo 'alias firecracker="cd /workspace/pkg/firecracker"' >> ~/.bashrc \
	&& echo 'alias firetest="./gow test -v ./pkg/firecracker/"' >> ~/.bashrc

# Run mod tidy to ensure dependencies
RUN ./gow mod tidy

# Default command
CMD ["/bin/bash"]
