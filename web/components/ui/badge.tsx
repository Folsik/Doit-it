import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";
const badgeVariants=cva("inline-flex w-fit items-center gap-1 rounded-lg border px-2.5 py-1 text-[11px] font-semibold",{
 variants:{variant:{default:"border-transparent bg-primary text-primary-foreground",secondary:"border-transparent bg-secondary text-secondary-foreground",outline:"border-border text-foreground",success:"border-transparent bg-emerald-50 text-emerald-700",warning:"border-transparent bg-amber-50 text-amber-700",destructive:"border-transparent bg-red-50 text-red-700"}},defaultVariants:{variant:"default"}});
function Badge({className,variant,...props}:React.ComponentProps<"span">&VariantProps<typeof badgeVariants>){return <span data-slot="badge" className={cn(badgeVariants({variant}),className)} {...props}/>;}
export {Badge,badgeVariants};
