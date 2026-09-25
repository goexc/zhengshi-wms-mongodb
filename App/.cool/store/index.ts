import { User, user } from "./user";
type Store = { user: User };
export function useStore(): Store { return { user }; }
export * from "./user";
