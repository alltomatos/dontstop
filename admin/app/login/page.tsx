import { redirect } from "next/navigation";
import { getSetupStatus } from "@/lib/admin-api";
import { LoginForm } from "./login-form";

export default async function LoginPage({
  searchParams
}: {
  searchParams: Promise<{ next?: string }>;
}) {
  const params = await searchParams;
  if (await getSetupStatus()) redirect("/onboarding");
  return (
    <main className="login-wrap">
      <LoginForm next={params.next?.startsWith("/") ? params.next : "/dashboard"} />
    </main>
  );
}
