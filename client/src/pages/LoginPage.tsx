import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { cn } from "@/lib/utils";
import { useAuth } from "../lib/auth";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";

const loginSchema = z.object({
  email: z
    .string()
    .min(1, "Email is required")
    .email("Please enter a valid email address"),
  password: z
    .string()
    .min(1, "Password is required")
    .min(8, "Password must be at least 8 characters"),
});

type LoginFormValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const { login, signup, error, clearError } = useAuth();
  const [mode, setMode] = useState<"login" | "signup">("login");

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    mode: "onSubmit",
    defaultValues: {
      email: "",
      password: "",
    },
  });

  const isSubmitting = form.formState.isSubmitting;

  async function onSubmit(data: LoginFormValues) {
    clearError();
    try {
      if (mode === "signup") {
        await signup(data.email.trim(), data.password);
      } else {
        await login(data.email.trim(), data.password);
      }
    } catch {
      // Error is set in auth context
    }
  }

  function handleModeSwitch(e: React.MouseEvent) {
    e.preventDefault();
    setMode((m) => (m === "login" ? "signup" : "login"));
    clearError();
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className={cn("flex flex-col w-full max-w-sm animate-slide-up")}>
        <Card className="px-4 py-4 sm:px-6 sm:py-6">
          <CardHeader className="gap-2">
            <CardTitle>
              {mode === "signup"
                ? "Create your account"
                : "Login to your account"}
            </CardTitle>
            <CardDescription>
              {mode === "signup"
                ? "Enter your email below to create your account"
                : "Enter your email below to login to your account"}
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-6">
            <form onSubmit={form.handleSubmit(onSubmit)}>
              <FieldGroup className="gap-6">
                <Controller
                  name="email"
                  control={form.control}
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor="email">Email</FieldLabel>
                      <Input
                        {...field}
                        id="email"
                        type="email"
                        placeholder="m@example.com"
                        autoComplete="email"
                        aria-invalid={fieldState.invalid}
                        onChange={(e) => {
                          field.onChange(e);
                          clearError();
                        }}
                      />
                      {fieldState.invalid && (
                        <FieldError errors={[fieldState.error]} />
                      )}
                    </Field>
                  )}
                />
                <Controller
                  name="password"
                  control={form.control}
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor="password">Password</FieldLabel>
                      <Input
                        {...field}
                        id="password"
                        type="password"
                        autoComplete={
                          mode === "signup"
                            ? "new-password"
                            : "current-password"
                        }
                        aria-invalid={fieldState.invalid}
                        onChange={(e) => {
                          field.onChange(e);
                          clearError();
                        }}
                      />
                      {fieldState.invalid && (
                        <FieldError errors={[fieldState.error]} />
                      )}
                    </Field>
                  )}
                />
                {error && (
                  <Alert variant="danger">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                <Field>
                  <Button
                    type="submit"
                    disabled={isSubmitting}
                    className="w-full mt-2"
                  >
                    {isSubmitting
                      ? "…"
                      : mode === "signup"
                        ? "Sign up"
                        : "Login"}
                  </Button>
                  <FieldDescription className="text-center pt-2">
                    {mode === "login" ? (
                      <>
                        Don&apos;t have an account?{" "}
                        <Button
                          type="button"
                          variant="link"
                          onClick={handleModeSwitch}
                          className="h-auto p-0 text-sm"
                        >
                          Sign up
                        </Button>
                      </>
                    ) : (
                      <>
                        Already have an account?{" "}
                        <Button
                          type="button"
                          variant="link"
                          onClick={handleModeSwitch}
                          className="h-auto p-0 text-sm"
                        >
                          Sign in
                        </Button>
                      </>
                    )}
                  </FieldDescription>
                </Field>
              </FieldGroup>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
