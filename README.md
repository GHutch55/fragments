![logo full](https://raw.githubusercontent.com/GHutch55/fragments/1c78300668f8fa217e017d9c2d8bf1aa48c13b18/frontend/src/assets/logoFull.svg)
# Fragments

A modern code snippet manager designed for developers who value organization and quick access to their code libraries.

## The Problem

As developers, we constantly Google the same patterns, copy code from old projects, and struggle to remember that perfect regex from last month. Existing solutions like GitHub Gists lack organization, and IDE snippets are locked to one editor.

## The Solution

Fragments provides a fast, searchable library for your code snippets with smart tagging, folder organization, and a clean web interface. Think of it as your personal Stack Overflow that learns from your actual code patterns.

## Features

- 🔍 **Full-text search** - Find snippets instantly across titles, content, and tags
- 🏷️ **Smart tagging** - Organize with flexible tagging system and auto-suggestions
- 📁 **Folder organization** - Hierarchical structure for logical grouping
- 🎨 **Syntax highlighting** - Support for 100+ programming languages
- ⚡ **Fast performance** - Built with Go for lightning-fast responses
- 🌐 **Clean web interface** - Intuitive UI built with modern web technologies

## Tech Stack

- **Backend**: Go with Chi router and SQLite
- **Frontend**: React + TypeScript (planned)
- **Hosting**: Railway (API) + Vercel (Frontend)

## Development

### Prerequisites
- Go 1.21 or higher
- Git

### Running Locally

```bash
cd api
go run main.go
```

Visit http://localhost:8080 to see the development server.

### Project Structure
```
fragments/
├── api/           # Go backend server
├── web/           # React frontend (coming soon)
└── docs/          # Documentation (coming soon)
```

## Roadmap

- [x] Basic server setup
- [x] Database schema and models
- [x] JWT-Based authentication
- [x] REST API endpoints
- [x] Search functionality
- [ ] Web interface
- [ ] Tag management
- [ ] Import/export features

## Contributing

This is currently a personal + learning project, but suggestions and feedback are welcome! Please open an issue to discuss any changes.

## License

MIT License - see [LICENSE](LICENSE) file for details.
