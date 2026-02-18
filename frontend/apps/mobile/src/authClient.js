import AsyncStorage from "@react-native-async-storage/async-storage";
import { createAuthClient } from "@raidx/shared";

const storage = {
  getItem(key) {
    return AsyncStorage.getItem(key);
  },
  setItem(key, value) {
    return AsyncStorage.setItem(key, value);
  },
  removeItem(key) {
    return AsyncStorage.removeItem(key);
  }
};

export const authClient = createAuthClient({
  baseUrl: "http://localhost:3000",
  storage
});
