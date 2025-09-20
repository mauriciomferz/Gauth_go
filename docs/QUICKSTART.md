
# GAuth Go Quickstart

---

## RFC 0111 Compliance & Legal Notice

GAuth implements the GiFo-RfC 0111 (GAuth) standard for AI power-of-attorney, delegation, and auditability. All protocol roles, flows, and exclusions are respected. See https://gimelfoundation.com for the full RFC.

**Exclusions:** GAuth MUST NOT include Web3, DNA-based identity, or decentralized auth logic. See RFC 0111 Section 2.

**Licensing:** Code is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents. See LICENSE, Apache 2.0, and referenced licenses for OAuth, OpenID Connect, and MCP.

**P*P Roles:** GAuth implements Power*Point roles (PEP, PDP, PIP, PAP, PVP) as defined in RFC 0111. See the Architecture Guide for details.

---

Welcome! This guide will help you get up and running with the GAuth Go library in minutes.

## 1. Install Go and Clone the Repo

```sh
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go
```

## 2. Run a Minimal Example

```sh
cd examples/minimal_server
# Run the minimal GAuth server example
go run main.go
```

You should see output showing a power of attorney, grant, and token being issued and an authorization check succeeding.

## 3. Explore the Library
- Library code: `pkg/gauth/`
- Demos: `examples/`
- API docs: `LIBRARY.md`, package-level `doc.go`

## 4. Next Steps
- Try modifying the example to use your own principal, agent, or scope.
- Explore more advanced examples in `examples/resilient/`.
- Read `LIBRARY.md` for API and extension guidance.

## 5. Need Help?
- See `README.md` and `CONTRIBUTING.md` for more info.
- Open an issue or discussion on GitHub for questions.

## 6. Next Steps: Extending and Integrating GAuth
- Extend GAuth: Add new grant types, token types, or audit event types by extending the relevant structs and interfaces in `pkg/gauth/`.
- Integrate with real apps: See `examples/web/gin_server.go` for web integration, or use GAuth in your own Go services.
- Contribute: Read `CONTRIBUTING.md` and join the community on GitHub Discussions.

## 7. Troubleshooting
- **Build errors about missing methods or types:** Ensure you are using the latest version of the codebase and that your Go environment is set up correctly.
- **Cannot find GAuth symbols:** Double-check your import paths and that you are importing from `pkg/gauth`.
- **Protocol errors or unexpected results:** Review the example code and ensure you are following the correct grant/token/authorization flow.
- **Still stuck?** Open an issue or discussion on GitHub for help.

## 8. How to Extend GAuth: Example

Suppose you want to add a new grant type:

1. Define your new grant struct in `pkg/gauth/types.go`:
   ```go
   type CustomGrant struct {
       GrantID    string
       CustomData string
       // ...other fields
   }
   ```
2. Add logic to issue and validate your new grant type in your application or by extending the GAuth server.
3. Use your new grant type in your own flows or examples.

For more, see `LIBRARY.md` and the advanced examples in `examples/advanced/`.

---

Happy hacking with GAuth!
