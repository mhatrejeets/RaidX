import { createAuthClient } from "@raidx/shared";

const storage = {
  getItem(key) {
    return Promise.resolve(localStorage.getItem(key));
  },
  setItem(key, value) {
    localStorage.setItem(key, value);
    return Promise.resolve();
  },
  removeItem(key) {
    localStorage.removeItem(key);
    return Promise.resolve();
  }
};

export const authClient = createAuthClient({
  baseUrl: "",
  storage
});