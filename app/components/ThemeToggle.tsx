"use client";

import { useTheme } from "next-themes";
import { useEffect, useState } from "react";

export default function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  // Avoid hydration mismatch
  useEffect(() => setMounted(true), []);
  if (!mounted) return <div className="w-[10.5rem] h-8" />;

  return (
    <select
      value={theme}
      onChange={(e) => setTheme(e.target.value)}
      className="text-sm rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 px-2 py-1 focus:outline-none focus:ring-2 focus:ring-gray-400 cursor-pointer"
      aria-label="Color theme"
    >
      <option value="light">Light</option>
      <option value="dark">Dark</option>
      <option value="system">Device Settings</option>
    </select>
  );
}
