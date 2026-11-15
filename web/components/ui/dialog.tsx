"use client";
import * as React from "react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
const Dialog=DialogPrimitive.Root;
const DialogTrigger=DialogPrimitive.Trigger;
const DialogClose=DialogPrimitive.Close;
function DialogContent({className,children,...props}:React.ComponentProps<typeof DialogPrimitive.Content>){
 return <DialogPrimitive.Portal><DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-slate-950/45 backdrop-blur-[2px] data-[state=open]:animate-in"/><DialogPrimitive.Content className={cn("fixed left-1/2 top-1/2 z-50 max-h-[90dvh] w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl border border-border bg-card p-6 text-card-foreground shadow-2xl outline-none",className)} {...props}>{children}<DialogPrimitive.Close className="absolute right-4 top-4 rounded-lg p-1.5 text-muted-foreground hover:bg-accent focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring" aria-label="Close dialog"><X size={17}/></DialogPrimitive.Close></DialogPrimitive.Content></DialogPrimitive.Portal>;
}
function DialogHeader({className,...props}:React.ComponentProps<"div">){return <div className={cn("mb-5 flex flex-col gap-2 pr-8",className)} {...props}/>;}
function DialogTitle({className,...props}:React.ComponentProps<typeof DialogPrimitive.Title>){return <DialogPrimitive.Title className={cn("text-xl font-bold",className)} {...props}/>;}
function DialogDescription({className,...props}:React.ComponentProps<typeof DialogPrimitive.Description>){return <DialogPrimitive.Description className={cn("text-sm text-muted-foreground",className)} {...props}/>;}
export {Dialog,DialogTrigger,DialogClose,DialogContent,DialogHeader,DialogTitle,DialogDescription};
