"use client";

import { createContext, useContext, useState, useEffect, ReactNode } from "react";
import { api } from "./api";
import type { Profile } from "./types";

interface UserContextType {
  user: Profile | null;
  loading: boolean;
  refetch: () => Promise<void>;
  updateUser: (data: Partial<Profile>) => void;
}

const UserContext = createContext<UserContextType | undefined>(undefined);

export function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchUser = async () => {
    try {
      const profile = await api.me();
      setUser(profile);
    } catch (e) {
      console.error("Failed to load user", e);
    } finally {
      setLoading(false);
    }
  };

  const refetch = async () => {
    setLoading(true);
    await fetchUser();
  };

  const updateUser = (data: Partial<Profile>) => {
    if (user) {
      setUser({ ...user, ...data });
    }
  };

  useEffect(() => {
    fetchUser();
  }, []);

  return (
    <UserContext.Provider value={{ user, loading, refetch, updateUser }}>
      {children}
    </UserContext.Provider>
  );
}

export function useUser() {
  const context = useContext(UserContext);
  if (context === undefined) {
    throw new Error("useUser must be used within a UserProvider");
  }
  return context;
}
