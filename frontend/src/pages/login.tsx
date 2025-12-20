import { useState } from "react";
import { authAPI } from "@/api";
import { useNavigate } from "react-router-dom";
import type { LoginInput } from "@/api";

import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import LogoFull from "@/assets/logoFull.svg";

export function LoginForm() {
  const [form, setForm] = useState<LoginInput>({
    username: "",
    password: "",
  });

  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    try {
      setLoading(true);
      const response = await authAPI.login({
        username: form.username,
        password: form.password,
      });

      localStorage.removeItem("authToken");
      localStorage.setItem("authToken", response.token);

      // Force a full page reload to clear all state
      window.location.href = "/dashboard";
    } catch (err: unknown) {
      if (typeof err === "string") {
        setError("Login Failed");
      } else if (err instanceof Error) {
        setError(err.message);
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col justify-center items-center h-screen bg-background">
      <img src={LogoFull} alt="Logo" className="w-80 h-20 mb-8" />
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Log in to Fragments</CardTitle>
          <CardDescription>
            Enter your info to log in to Fragments
          </CardDescription>
          <CardAction>
            <Button variant="default" onClick={() => navigate("/register")}>
              Sign Up
            </Button>
          </CardAction>
        </CardHeader>

        <CardContent>
          <form className="space-y-6" onSubmit={handleSubmit}>
            <div className="space-y-2">
              <Label htmlFor="username">Username *</Label>
              <Input
                id="username"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password *</Label>
              <Input
                id="password"
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                required
              />
            </div>

            {error && <p className="text-red-500 text-sm">{error}</p>}

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Logging In..." : "Log In"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
