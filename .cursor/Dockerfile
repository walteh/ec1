FROM ubuntu:24.10

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
	&& go install github.com/go-delve/delve/cmd/dlv@latest \
	&& go install github.com/vektra/mockery/v2@latest

# Set working directory
WORKDIR /workspace

# Copy project and ensure gow is available
COPY . .
RUN chmod +x ./gow 2> /dev/null || echo "gow will be available when needed"

# Set up development aliases in bashrc
RUN echo 'export PATH="/workspace:$PATH"' >> ~/.bashrc \
	&& echo 'alias gow="./gow"' >> ~/.bashrc \
	&& echo 'alias gowtest="./gow test -function-coverage -v"' >> ~/.bashrc \
	&& echo 'alias firecracker="cd /workspace/pkg/firecracker"' >> ~/.bashrc \
	&& echo 'alias vmm="cd /workspace/pkg/vmm"' >> ~/.bashrc \
	&& echo 'alias bootloader="cd /workspace/pkg/bootloader"' >> ~/.bashrc \
	&& echo 'alias performance="cd /workspace/pkg/testing/tstream"' >> ~/.bashrc \
	&& echo 'alias fulltest="./gow test -function-coverage ./..."' >> ~/.bashrc \
	&& echo 'alias benchmark-firecracker="./gow test -bench=. ./pkg/firecracker/"' >> ~/.bashrc

# Run mod tidy to ensure dependencies
RUN ./gow mod tidy

# Default command
CMD ["/bin/bash"]
