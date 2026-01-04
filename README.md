# ![Fragments Logo](https://raw.githubusercontent.com/GHutch55/fragments/1c78300668f8fa217e017d9c2d8bf1aa48c13b18/frontend/src/assets/logoFull.svg)

A full-stack web application that enables developers to securely store, organize, and efficiently search personal code snippets.

[Live Demo:](https://fragments-7gas.onrender.com/) 

---

## Overview

Developers often reuse common patterns, utility functions, and reference snippets across projects. These snippets are frequently scattered across old repositories, notes, or browser bookmarks, making them difficult to find when needed.

Fragments is a personal code snippet manager designed to centralize these patterns in a single, searchable web application. The project was built to practice backend system design, database modeling, and modern frontend development, with an emphasis on correctness, security, and performance.

---

## Core Features

- Full-text search using PostgreSQL for efficient lookup across snippet titles and content
- Hierarchical folder organization for logical grouping of snippets
- Secure multi-user authentication with complete data isolation
- RESTful API with ownership validation on all protected endpoints
- Code editor with syntax highlighting for 12+ programming languages
- Responsive web interface built with React and TypeScript

---

## Tech Stack

Backend:

- Go (Chi Router)
- PostgreSQL (Supabase)
- JWT-based authentication
- bcrypt password hashing
- Middleware for validation, CORS, and rate limiting

Frontend:

- React
- TypeScript
- Monaco Editor for code editing and syntax highlighting
- React Context for state management

Infrastructure:

- Database: PostgreSQL (Supabase)
- Deployment: Render

---

## Engineering Highlights

- Designed and implemented 20+ RESTful API endpoints following standard REST conventions
- Enforced authentication, authorization, and ownership checks on all CRUD operations
- Used transactions to ensure correctness for multi-step database operations
- Structured backend code with clear separation between routing, business logic, and data access
- Focused on query efficiency and low-latency backend responses

---

## Purpose

Fragments was built as a backend-focused portfolio project to demonstrate practical software engineering skills relevant to internship and early-career roles, including:

- Designing and building RESTful services in Go
- Implementing authentication and authorization correctly
- Working with relational databases and SQL
- Reasoning about data integrity, security, and performance
- Integrating a typed frontend with a backend API

---

## Contributing

This is currently a personal project, but feedback and suggestions are welcome. Feel free to open an issue to discuss improvements.

---

## License

MIT License - see [LICENSE](LICENSE) file for details.
