LOGO

# AzHexGate

AzHexGate is a self‑hosted, Azure‑native reverse tunneling platform designed to expose any local application to the internet securely, reliably, and with zero friction. Think of it as a Hextech‑powered gateway: your localhost becomes globally reachable through your own Azure infrastructure — no third‑party relay services, no black boxes, full control.

This project is currently **Work in Progress (WIP)**.  
The architecture is defined and development is progressing through small, reviewable steps powered by GitHub Copilot and automated CI.

---

## 🚀 What AzHexGate Does

- Creates ephemeral public URLs like `https://12345678.azhexgate.com`
- Forwards traffic through Azure Relay Hybrid Connections
- Requires **no inbound ports** on the user’s machine
- Runs entirely on Azure resources you own
- Provides a clean CLI experience:
  ```bash
  azhexgate start --port 3000
  ```

---

## 🧩 High‑Level Architecture

AzHexGate is composed of:

- **Local Client (Go CLI)**  
  Runs on the user’s machine, connects outbound to Azure Relay, and forwards traffic to `localhost`.

- **Cloud Gateway (Go, Azure App Service)**  
  Public entrypoint that routes incoming HTTPS traffic to the correct tunnel.

- **Management API**  
  Issues tunnel metadata, subdomains, and short‑lived Relay tokens.

- **Azure Infrastructure (Bicep)**  
  Relay namespace, App Service, DNS, Key Vault, Managed Identity, and observability.

For full details, see:  
`docs/architecture.md`

---

## 🛠️ Project Status

AzHexGate is under active development.  
The roadmap includes:

- CLI scaffolding  
- Cloud Gateway skeleton  
- Management API  
- Azure Relay integration  
- Infrastructure deployment  
- End‑to‑end tunnel tests  

All work is tracked through small, focused GitHub issues and PRs.

---

## 🤝 Contributing

Contributions are welcome once the core scaffolding is complete.  
The project uses:

- Small, atomic PRs  
- Automated CI (build + tests)  
- Architecture‑first development  
- GitHub Projects for planning  

---

## 📜 License

MIT License.

---

Stay tuned — the Hextech gates are opening soon.
