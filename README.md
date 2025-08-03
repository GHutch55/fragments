# Fragments

A modern code snippet manager designed for developers who value organization and quick access to their code libraries.

## Features

- 🔍 **Full-text search** - Find snippets instantly across titles, content, and tags
- 🏷️ **Smart tagging** - Organize with flexible tagging system and auto-suggestions
- 📁 **Folder organization** - Hierarchical structure for logical grouping
- 🎨 **Syntax highlighting** - Support for 100+ programming languages
- ⚡ **Fast performance** - Built with Go for lightning-fast responses
- 🌐 **Clean web interface** - Intuitive UI built with modern web technologies

## Tech Stack

- **Backend**: Go with standard library HTTP server
- **Database**: PostgreSQL (planned)
- **Frontend**: React + TypeScript (planned)

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
- [ ] Database schema and models
- [ ] REST API endpoints
- [ ] Search functionality
- [ ] Web interface
- [ ] Tag management
- [ ] Import/export features

## Contributing

This is currently a personal project, but suggestions and feedback are welcome! Please open an issue to discuss any changes.

## License

MIT License - see LICENSE file for details.
