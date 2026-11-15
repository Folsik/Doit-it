"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { Bell, CheckCircle2, ChevronRight, Globe2, Languages, Moon, ShieldCheck, UserRound, Wallet } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";

type User = { id:string; full_name:string; email:string; country_code:string; preferred_currency:string; language_code:string };
type Settings = {address_line1:string;address_line2:string;city:string;postal_code:string;residence_country:string;notifications_email:boolean;notifications_push:boolean;notifications_security:boolean;appearance:"system"|"light"|"dark"};
type Section = "personal"|"address"|"currency"|"language"|"notifications"|"appearance"|null;
const defaults:Settings={address_line1:"",address_line2:"",city:"",postal_code:"",residence_country:"",notifications_email:true,notifications_push:true,notifications_security:true,appearance:"system"};
const languages=[{code:"en",name:"English"},{code:"de",name:"Deutsch"},{code:"fr",name:"Français"},{code:"es",name:"Español"}] as const;
const translations={
 en:{title:"Your profile",subtitle:"The details that make your account yours.",personal:"Personal information",address:"Country and address",currency:"Currency",language:"Language",notifications:"Notifications",appearance:"App preferences",details:"Personal details",prefs:"Preferences",edit:"Edit profile",save:"Save changes",cancel:"Cancel"},
 de:{title:"Dein Profil",subtitle:"Deine persönlichen Daten und Einstellungen.",personal:"Persönliche Daten",address:"Land und Adresse",currency:"Währung",language:"Sprache",notifications:"Benachrichtigungen",appearance:"App-Einstellungen",details:"Persönliche Angaben",prefs:"Einstellungen",edit:"Profil bearbeiten",save:"Speichern",cancel:"Abbrechen"},
 fr:{title:"Votre profil",subtitle:"Vos informations et préférences personnelles.",personal:"Informations personnelles",address:"Pays et adresse",currency:"Devise",language:"Langue",notifications:"Notifications",appearance:"Préférences",details:"Informations personnelles",prefs:"Préférences",edit:"Modifier le profil",save:"Enregistrer",cancel:"Annuler"},
 es:{title:"Tu perfil",subtitle:"Tus datos y preferencias personales.",personal:"Información personal",address:"País y dirección",currency:"Moneda",language:"Idioma",notifications:"Notificaciones",appearance:"Preferencias",details:"Datos personales",prefs:"Preferencias",edit:"Editar perfil",save:"Guardar cambios",cancel:"Cancelar"},
};
type Locale=keyof typeof translations;
async function api<T>(path:string,method:"GET"|"PUT"|"PATCH"|"POST"="GET",body?:unknown):Promise<T>{
 const response=await fetch("/api/banking"+path,{method,credentials:"same-origin",headers:{"Content-Type":"application/json"},body:body===undefined?undefined:JSON.stringify(body),cache:"no-store"});
 const result=await response.json().catch(()=>({}));
 if(!response.ok) throw Object.assign(new Error(result?.error?.message??(response.status===401?"Session expired. Please sign in again.":`Profile request failed (HTTP ${response.status})`)),{status:response.status});
 return result as T;
}
const field="space-y-2";
export default function ProfileSettings(){
 const [user,setUser]=useState<User|null>(null),[settings,setSettings]=useState<Settings>(defaults);
 const [section,setSection]=useState<Section>(null),[loading,setLoading]=useState(true),[busy,setBusy]=useState(false),[unauthorized,setUnauthorized]=useState(false);
 const [error,setError]=useState(""),[notice,setNotice]=useState("");
 const locale:Locale=(user?.language_code&&user.language_code in translations?user.language_code:"en") as Locale;
 const t=translations[locale];
 useEffect(()=>{
  const media=window.matchMedia("(prefers-color-scheme: dark)");
  const apply=()=>{document.documentElement.dataset.doitTheme=settings.appearance==="system"?(media.matches?"dark":"light"):settings.appearance;};
  apply();media.addEventListener("change",apply);
  return ()=>media.removeEventListener("change",apply);
 },[settings.appearance]);
 useEffect(()=>{
  let active=true;
  Promise.all([api<User>("/v1/me"),api<Settings>("/v1/me/settings")]).then(([u,s])=>{if(active){setUser(u);setSettings(s);document.documentElement.lang=u.language_code in translations?u.language_code:"en";}})
  .catch((e:Error&{status?:number})=>{if(active){setUnauthorized(e.status===401);setError(e.message);}})
  .finally(()=>{if(active)setLoading(false)});
  return ()=>{active=false};
 },[]);
 async function save(method:"PUT"|"PATCH",path:string,body:unknown){
  setBusy(true);setError("");setNotice("");
  try{
   if(path==="/v1/me"){const u=await api<User>(path,method,body);setUser(u);document.documentElement.lang=u.language_code in translations?u.language_code:"en";}
   else {const s=await api<Settings>(path,method,body);setSettings(s);}
   setNotice("Changes saved successfully");setSection(null);
  }catch(e){setError(e instanceof Error?e.message:"Could not save changes");}
  finally{setBusy(false);}
 }
 async function requestVerification(){
  setBusy(true);setError("");setNotice("");
  try{await api("/v1/me/identity/reverification","POST",{});}
  catch(e){setNotice(e instanceof Error?e.message:"Veriff verification has not been configured yet.");setSection(null);}
  finally{setBusy(false);}
 }
 if(loading)return <Card className="p-8 text-sm text-muted-foreground" role="status">Loading account settings…</Card>;
 if(unauthorized)return <Card className="space-y-4 p-8"><h2 className="text-xl font-bold">Sign in to manage your profile</h2><p className="text-sm text-muted-foreground">Your settings are only available after authentication.</p><Button asChild><Link href="/auth">Sign in or create account</Link></Button></Card>;
 if(!user)return <Card className="space-y-3 p-8" role="alert"><p className="text-sm text-destructive">{error||"Could not load your profile"}</p><Button variant="outline" onClick={()=>window.location.reload()}>Retry</Button></Card>;
 const initials=user.full_name?user.full_name.trim().split(/\s+/).slice(0,2).map(x=>x[0]).join("").toUpperCase():"U";
 const sections:{id:Exclude<Section,null>;title:string;subtitle:string;icon:typeof UserRound}[]=[
  {id:"personal",title:t.personal,subtitle:user.full_name||"Legal details",icon:UserRound},
  {id:"address",title:t.address,subtitle:[settings.residence_country||user.country_code,settings.city].filter(Boolean).join(" · ")||"Set up your address",icon:Globe2},
  {id:"currency",title:t.currency,subtitle:"Euro (EUR)",icon:Wallet},
  {id:"language",title:t.language,subtitle:languages.find(l=>l.code===user.language_code)?.name??"English",icon:Languages},
  {id:"notifications",title:t.notifications,subtitle:"Manage alerts and updates",icon:Bell},
  {id:"appearance",title:t.appearance,subtitle:settings.appearance==="system"?"Follow system appearance":settings.appearance+" mode",icon:Moon}
 ];
 const row=(item:typeof sections[number])=>{const Icon=item.icon;return <button key={item.id} onClick={()=>{setSection(item.id);setError("");}} type="button" className="flex w-full items-center gap-4 border-b border-border py-5 text-left transition-colors last:border-0 hover:text-primary focus-visible:rounded-lg focus-visible:outline-2 focus-visible:outline-ring">
  <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-secondary text-primary"><Icon size={19}/></span>
  <span className="flex-1"><strong className="block text-sm font-semibold text-foreground">{item.title}</strong><small className="mt-1 block text-xs text-muted-foreground">{item.subtitle}</small></span><ChevronRight size={18} className="text-muted-foreground"/>
 </button>};
 const onSubmit=(e:FormEvent,method:"PUT"|"PATCH",path:string,body:unknown)=>{e.preventDefault();void save(method,path,body)};
 return <div className="space-y-5">
  <div className="mb-7 flex items-start justify-between gap-4"><div><p className="mb-2 text-[10px] font-bold uppercase tracking-[.18em] text-muted-foreground">Your space</p><h1 className="font-display text-3xl font-extrabold tracking-tight sm:text-4xl">{t.title}</h1><p className="mt-3 text-sm text-muted-foreground">{t.subtitle}</p></div><Button variant="outline" asChild><Link href="/?view=Security"><ShieldCheck size={17}/> Security</Link></Button></div>
  <Card className="flex flex-wrap items-center gap-5 p-6"><div className="grid size-16 place-items-center rounded-full bg-gradient-to-br from-blue-100 to-blue-300 text-xl font-bold text-blue-700">{initials}</div><div className="min-w-0 flex-1"><h2 className="text-lg font-bold">{user.full_name||"Account holder"}</h2><p className="mt-1 text-xs text-muted-foreground">{user.email||"No email added"}</p><Badge variant="outline" className="mt-2">Prototype account</Badge></div><Button variant="outline" size="sm" onClick={()=>setSection("personal")}>{t.edit}</Button></Card>
  <Card><CardHeader><CardTitle>{t.details}</CardTitle></CardHeader><CardContent>{sections.slice(0,4).map(row)}</CardContent></Card>
  <Card><CardHeader><CardTitle>{t.prefs}</CardTitle></CardHeader><CardContent>{sections.slice(4).map(row)}</CardContent></Card>
  {notice&&<p className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800" role="status">{notice}</p>}
  <Dialog open={section!==null} onOpenChange={o=>{if(!o&&!busy)setSection(null)}}>
   <DialogContent>
    <DialogHeader><DialogTitle>{sections.find(x=>x.id===section)?.title}</DialogTitle><DialogDescription>Manage your account preferences. Changes are saved securely.</DialogDescription></DialogHeader>
    {error&&<p className="rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700" role="alert">{error}</p>}
    {section==="personal"&&<div className="space-y-5"><div className={field}><Label>Legal name</Label><Input disabled readOnly value={user.full_name}/></div><div className={field}><Label>Email</Label><Input disabled readOnly value={user.email||""}/></div><div className={field}><Label>Identity country</Label><Input disabled readOnly value={user.country_code||"Not set"}/></div><p className="text-xs leading-5 text-muted-foreground">Legal name, date of birth, nationality and residence country changes require Veriff re-verification.</p><Button disabled={busy} onClick={()=>void requestVerification()}><ShieldCheck size={16}/> Request identity verification</Button></div>}
    {section==="address"&&<form className="space-y-4" onSubmit={e=>onSubmit(e,"PUT","/v1/me/address",{address_line1:settings.address_line1,address_line2:settings.address_line2,city:settings.city,postal_code:settings.postal_code})}>
      <div className={field}><Label>Residence country (Veriff required)</Label><Input value={settings.residence_country||user.country_code||"Not set"} disabled readOnly/></div>
      <Button size="sm" type="button" variant="outline" disabled={busy} onClick={()=>void requestVerification()}>Request country change</Button>
      {([{key:"address_line1",label:"Street address",maxLength:200},{key:"address_line2",label:"Apartment / building",maxLength:200},{key:"city",label:"City",maxLength:120},{key:"postal_code",label:"Postal code",maxLength:32}] as const).map(x=><div className={field} key={x.key}><Label htmlFor={x.key}>{x.label}</Label><Input id={x.key} maxLength={x.maxLength} value={settings[x.key]} onChange={e=>setSettings(s=>({...s,[x.key]:e.target.value}))}/></div>)}
      <Button disabled={busy} className="w-full">{busy?"Saving…":t.save}</Button></form>}
    {section==="currency"&&<div className="space-y-4"><Label>Account currency</Label><Select disabled value="EUR"><option value="EUR">Euro (EUR)</option></Select><p className="text-xs leading-5 text-muted-foreground">EUR is currently the only supported currency. Currency changes do not convert an account balance.</p><Button variant="outline" onClick={()=>setSection(null)}>Close</Button></div>}
    {section==="language"&&<form className="space-y-4" onSubmit={e=>onSubmit(e,"PATCH","/v1/me",{language_code:user.language_code})}><div className={field}><Label htmlFor="profile-language">Language</Label><Select id="profile-language" value={user.language_code} onChange={e=>setUser(u=>u?{...u,language_code:e.target.value}:u)}>{languages.map(l=><option key={l.code} value={l.code}>{l.name}</option>)}</Select></div><p className="text-xs text-muted-foreground">Profile labels are localized; full dashboard translation is scheduled separately.</p><Button className="w-full" disabled={busy}>{busy?"Saving…":t.save}</Button></form>}
    {section==="notifications"&&<form className="space-y-5" onSubmit={e=>onSubmit(e,"PUT","/v1/me/notifications",{email:settings.notifications_email,push:settings.notifications_push,security:true})}>
      {([{key:"notifications_email",label:"Email notifications"},{key:"notifications_push",label:"Push notifications"}] as const).map(x=><div key={x.key} className="flex items-center justify-between gap-4"><Label htmlFor={x.key}>{x.label}</Label><Switch id={x.key} checked={settings[x.key]} onCheckedChange={v=>setSettings(s=>({...s,[x.key]:v}))}/></div>)}
      <div className="flex items-center justify-between gap-4"><Label>Security notifications (required)</Label><Switch checked disabled/></div>
      <p className="text-xs leading-5 text-muted-foreground">Notification delivery integrations are not configured. These settings are preferences only.</p><Button className="w-full" disabled={busy}>{busy?"Saving…":t.save}</Button></form>}
    {section==="appearance"&&<form className="space-y-4" onSubmit={e=>onSubmit(e,"PUT","/v1/me/appearance",{appearance:settings.appearance})}><div className={field}><Label htmlFor="appearance">Appearance</Label><Select id="appearance" value={settings.appearance} onChange={e=>setSettings(s=>({...s,appearance:e.target.value as Settings["appearance"]}))}><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></Select></div><Button className="w-full" disabled={busy}>{busy?"Saving…":t.save}</Button></form>}
   </DialogContent>
  </Dialog>
 </div>;
}
