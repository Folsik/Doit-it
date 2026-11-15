"use client";
import * as React from "react";
import * as SwitchPrimitive from "@radix-ui/react-switch";
import { cn } from "@/lib/utils";
function Switch({className,...props}:React.ComponentProps<typeof SwitchPrimitive.Root>){return <SwitchPrimitive.Root className={cn("peer relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full bg-slate-300 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 data-[state=checked]:bg-primary disabled:cursor-not-allowed disabled:opacity-50",className)} {...props}><SwitchPrimitive.Thumb className="pointer-events-none block size-5 translate-x-0.5 rounded-full bg-white shadow-sm transition-transform data-[state=checked]:translate-x-[22px]"/></SwitchPrimitive.Root>;}
export {Switch};
