import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Route, Routes, Navigate } from "react-router-dom";
import "./index.css";
import { RegisterForm } from "./pages/register.tsx";
import { LoginForm } from "./pages/login.tsx";
import { FileTreeProvider } from "./hooks/fileTreeProvider.tsx";
import { Dashboard } from "./pages/dashboard.tsx";
import { EditorPage } from "./pages/editor.tsx";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <FileTreeProvider>
        <Routes>
          <Route path="/" element={<Navigate to="/login" replace />} />
          <Route path="/register" element={<RegisterForm />} />
          <Route path="/login" element={<LoginForm />} />
          <Route path="/editor" element={<EditorPage />} />
          <Route path="/editor/:id" element={<EditorPage />} />
          <Route path="/dashboard" element={<Dashboard />} />
        </Routes>
      </FileTreeProvider>
    </BrowserRouter>
  </React.StrictMode>,
);
