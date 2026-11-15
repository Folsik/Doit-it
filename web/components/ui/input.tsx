import * as React from "react";
import { cn } from "@/lib/utils";
function Input({className,type,...props}:React.ComponentProps<"input">){return <input data-slot="input" type={type} className={cn("flex h-11 w-full rounded-xl border border-input bg-background px-3.5 py-2 text-sm text-foreground shadow-xs outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/25 disabled:cursor-not-allowed disabled:opacity-50",className)} {...props}/>;}
export {Input};
