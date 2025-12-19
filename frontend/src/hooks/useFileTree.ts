import { useContext } from "react";
import { FileTreeContext } from "./fileTreeContext";
import type { FileTreeContextType } from "./fileTreeContext";

export function useFileTree(): FileTreeContextType {
  const context = useContext(FileTreeContext);
  if (!context) {
    throw new Error("useFileTree must be used within FileTreeProvider");
  }
  return context;
}
