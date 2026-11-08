import { LoginForm } from "./login-form";

export default async function LoginPage({
  searchParams
}: {
  searchParams: Promise<{ next?: string }>;
}) {
  const params = await searchParams;
  return (
    <main className="login-wrap">
      <LoginForm next={params.next?.startsWith("/") ? params.next : "/dashboard"} />
    </main>
  );
}
