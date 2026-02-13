import { create } from "zustand";

type AuthStore = {
  isAuthenticated: boolean;
  token: string | null;
  login: (token: string) => void;
  logout: () => void;
};

export const useAuthStore = create<AuthStore>((set) => ({
  isAuthenticated: false,
  token: null,
  login: (token: string) => {
    localStorage.setItem("auth_token", token); // Persist token
    set({ isAuthenticated: true, token });
  },
  logout: () => {
    localStorage.removeItem("auth_token");
    set({ isAuthenticated: false, token: null });
  },
}));
