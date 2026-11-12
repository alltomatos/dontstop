import type { Metadata } from "next";
import { cookies } from "next/headers";
import "./globals.css";
import { I18nProvider } from "@/components/console/i18n-provider";
import { resolveLocale, STORAGE_KEY } from "@/lib/i18n";

export const metadata: Metadata = {
  title: "DonStop IA - Console",
  description: "Console administrativo do gateway corporativo DonStop IA."
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const store = await cookies();
  const initialLocale = resolveLocale(store.get(STORAGE_KEY)?.value);
  return (
    <html lang={initialLocale} suppressHydrationWarning>
      <body>
        <I18nProvider initialLocale={initialLocale}>{children}</I18nProvider>
      </body>
    </html>
  );
}
