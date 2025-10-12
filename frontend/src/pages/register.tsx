import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { authAPI } from "@/api";
import type { RegisterInput } from "@/api";

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

interface RegisterFormState extends RegisterInput {
  confirmPassword: string;
}

function validatePassword(password: string): string | null {
  if (password.length < 12) {
    return "Password must be at least 12 characters long.";
  }
  if (!/[A-Z]/.test(password)) {
    return "Password must include at least one uppercase letter.";
  }
  if (!/[a-z]/.test(password)) {
    return "Password must include at least one lowercase letter.";
  }
  if (!/[0-9]/.test(password)) {
    return "Password must include at least one number.";
  }
  if (!/[^A-Za-z0-9]/.test(password)) {
    return "Password must include at least one special character.";
  }
  return null;
}

export function RegisterForm() {
  const [form, setForm] = useState<RegisterFormState>({
    username: "",
    password: "",
    confirmPassword: "",
  });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const navigate = useNavigate();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (form.password !== form.confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    if (validatePassword(form.password)) {
      setError("Password does not meet requirements");
      return;
    }

    try {
      setLoading(true);
      const user = await authAPI.register({
        username: form.username,
        password: form.password,
      });
      console.log("Registered:", user);
      navigate("/login")
    } catch (err: unknown) {
      if (typeof err === "string") {
        setError("Registration Failed");
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
          <CardTitle>Sign up for Fragments</CardTitle>
          <CardDescription>
            Enter your info to sign up for Fragments
          </CardDescription>
          <CardAction>
            <Button variant="default" onClick={() => navigate("/login")}>Log In</Button>
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
                onChange={(e) => {
                  setForm({ ...form, password: e.target.value });
                  setPasswordError(validatePassword(e.target.value));
                }}
                required
              />
              <p className="text-sm text-muted-foreground">
                Must be at least 12 characters with uppercase, lowercase,
                number, and special character
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Confirm Password *</Label>
              <Input
                id="confirmPassword"
                type="password"
                value={form.confirmPassword}
                onChange={(e) =>
                  setForm({ ...form, confirmPassword: e.target.value })
                }
                required
              />
            </div>

            {passwordError && (
              <p className="text-red-500 text-sm">{passwordError}</p>
            )}
            {error && <p className="text-red-500 text-sm">{error}</p>}

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Creating Account..." : "Create Account"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
