import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
const allowed: Record<string, readonly string[]> = {
  "GET": ["/v1/me","/v1/me/settings"],
  "PATCH":["/v1/me"],
  "PUT":["/v1/me/address","/v1/me/notifications","/v1/me/appearance"],
  "POST":["/v1/me/identity/reverification"]
};
async function handler(request:NextRequest,{params}:{params:Promise<{path:string[]}>}) {
  const method=request.method;
  const path="/"+(await params).path.join("/");
  if (!allowed[method]?.includes(path)) return NextResponse.json({error:{message:"Endpoint not allowed"}},{status:404});
  if (method!=="GET") {
    const origin=request.headers.get("origin");
    let valid=false;
    try { const parsed=new URL(origin ?? "");valid=(parsed.protocol==="http:" || parsed.protocol==="https:") && parsed.host===request.headers.get("host"); }catch{}
    if (!valid) return NextResponse.json({error:{message:"Invalid origin"}},{status:403});
  }
  const token=request.cookies.get("doit_session")?.value;
  if (!token) return NextResponse.json({error:{message:"Please sign in"}},{status:401});
  const url=(process.env.API_INTERNAL_URL??"http://127.0.0.1:8080")+path;
  const headers:Record<string,string>={Authorization:"Bearer "+token};
  if(method!=="GET") headers["Content-Type"]="application/json";
  let body: string|undefined;
  if(method!=="GET") {
    body=await request.text();
    if(body.length>8192) return NextResponse.json({error:{message:"Request too large"}},{status:413});
  }
  try{
    const upstream=await fetch(url,{method,headers,body,cache:"no-store",signal:AbortSignal.timeout(10000)});
    const response= new NextResponse(await upstream.text(),{status:upstream.status,headers:{"Content-Type":"application/json","Cache-Control":"no-store"}});
    if(upstream.status===401) response.cookies.delete("doit_session");
    return response;
  }catch{return NextResponse.json({error:{message:"Bank API is unavailable"}},{status:503});}
}
export const GET=handler;
export const PATCH=handler;
export const PUT=handler;
export const POST=handler;
