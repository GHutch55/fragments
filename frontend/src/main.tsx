import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import "./index.css";
import { RegisterForm } from "./pages/register.tsx";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/register" element={<RegisterForm />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
);
