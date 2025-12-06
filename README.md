# Terminal Wrapped 🎄✨
<img src="wrapped.png" alt="Alt text" width="800"/>

A CLI tool to visualize your terminal usage history, written in Go.

![Terminal Wrapped](wrapped.png)

## Features

- 📊 **Interactive TUI**: Built with Bubble Tea for a smooth experience.
- 📈 **Detailed Stats**: Analyze your top commands, daily activity, and more.
- 🤖 **AI Roasts**: Get a witty roast of your terminal habits (powered by Nebius AI).
- 🐳 **Self-Hosted Proxy**: Securely host the AI proxy on your own infrastructure.

## Installation

### From Source

```bash
go install github.com/rgrouls/terminal-wrapped/cmd/terminal-wrapped@latest
```

### From Binary

Download the latest release from the [Releases](https://github.com/rgrouls/terminal-wrapped/releases) page.

## Usage

Run the tool:

```bash
terminal-wrapped
```

### AI Features (Optional)

To enable the AI features, you need to run the proxy service.

#### 1. Deploy the Proxy
### 3. Deploy Proxy (Optional)

If you want to host the AI proxy on a remote server (e.g., a VM):

1.  **Configure**:
    - Edit `proxy/.env` locally with your Nebius API key.
    - Edit `deploy.sh` to set your `SERVER_IP` and `USER`.

2.  **Deploy**:
    ```bash
    ./deploy.sh
    ```
    This script will copy the necessary files to your server and start the proxy using Docker Compose.

3.  **Verify**:
    ```bash
    curl http://<your-server-ip>:8080/health
    ```

#### 2. Configure the App

By default, the app connects to `http://localhost:8080/roast`. If you deployed the proxy to a remote server, set the `TERMINAL_WRAPPED_PROXY` environment variable:

```bash
export TERMINAL_WRAPPED_PROXY="http://your-server-ip:8080/roast"
./terminal-wrapped
```

For example, if your server IP is `145.38.190.202`:
```bash
export TERMINAL_WRAPPED_PROXY="http://145.38.190.202:8080/roast"
./terminal-wrapped
```

If running locally:
```bash
# Ensure the proxy is running locally
cd proxy
export NEBIUS_API_KEY="your-key"
go run main.go
# OR
docker-compose up
```
