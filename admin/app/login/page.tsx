import { redirect } from "next/navigation";
import { getSetupStatus } from "@/lib/admin-api";
import { LoginForm } from "./login-form";

export const dynamic = "force-dynamic";

function safeNextPath(value: string | undefined): string {
  if (!value || !value.startsWith("/")) return "/dashboard";
  if (value.startsWith("//") || value.startsWith("/\\")) return "/dashboard";
  return value;
}

export default async function LoginPage({
  searchParams
}: {
  searchParams: Promise<{ next?: string }>;
}) {
  const params = await searchParams;
  if (await getSetupStatus()) redirect("/onboarding");
  return (
    <main className="login-wrap">
      <LoginForm next={safeNextPath(params.next)} />
    </main>
  );
}
