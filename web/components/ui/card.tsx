import * as React from "react";
import { cn } from "@/lib/utils";
function Card({className,...props}:React.ComponentProps<"div">){return <div data-slot="card" className={cn("rounded-[22px] border border-border bg-card text-card-foreground shadow-[0_7px_30px_rgba(30,52,86,0.035)]",className)} {...props}/>;}
function CardHeader({className,...props}:React.ComponentProps<"div">){return <div data-slot="card-header" className={cn("flex flex-col gap-2 px-6 pt-6",className)} {...props}/>;}
function CardTitle({className,...props}:React.ComponentProps<"h3">){return <h3 data-slot="card-title" className={cn("text-base font-bold tracking-tight",className)} {...props}/>;}
function CardDescription({className,...props}:React.ComponentProps<"p">){return <p data-slot="card-description" className={cn("text-sm text-muted-foreground",className)} {...props}/>;}
function CardContent({className,...props}:React.ComponentProps<"div">){return <div data-slot="card-content" className={cn("px-6 pb-6",className)} {...props}/>;}
function CardFooter({className,...props}:React.ComponentProps<"div">){return <div data-slot="card-footer" className={cn("flex items-center px-6 pb-6",className)} {...props}/>;}
export {Card,CardHeader,CardTitle,CardDescription,CardContent,CardFooter};
