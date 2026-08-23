import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import { I18nProvider } from "@/lib/i18n";

// Inter across every weight for all UI text — chosen specifically because
// it has solid Cyrillic coverage, which matters here since 4 of the 5
// supported locales (en) use Cyrillic script. JetBrains Mono
// is reserved for SQL/code surfaces, marking those out from the rest of
// the UI by typeface rather than just size.
const inter = Inter({ subsets: ["latin", "cyrillic"], variable: "--font-inter" });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-mono" });

export const metadata: Metadata = {
  title: "StickStock",
  description: "Self-hosted BI: connect, query, visualize.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`dark ${inter.variable} ${mono.variable}`}>
      <body className="font-body bg-base-950 text-slate-100 antialiased">
        <I18nProvider>{children}</I18nProvider>
      </body>
    </html>
  );
}
