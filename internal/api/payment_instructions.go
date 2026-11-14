package api

import (
 "crypto/sha256"
 "bytes"
 "encoding/json"
 "net/http"
 "strings"
)

type paymentInstructionRequest struct {
 SourceAccountID string `json:"source_account_id"`
 Rail string `json:"rail"`
 AmountMinor int64 `json:"amount_minor"`
 Currency string `json:"currency"`
 RecipientCurrency string `json:"recipient_currency"`
 IdempotencyKey string `json:"idempotency_key"`
 Recipient json.RawMessage `json:"recipient"`
}

func (s *Server) createPaymentInstruction(w http.ResponseWriter,r *http.Request) {
 var v paymentInstructionRequest
 if !decode(w,r,&v){return}
 switch v.Rail {case "sepa","sepa_instant","local_bank","international_bank","swift","card":default:writeError(w,400,"validation_error","Unsupported payment rail");return}
 if !validUUID(v.SourceAccountID)||v.AmountMinor<=0||v.Currency!="EUR"||v.RecipientCurrency!="EUR"||len(v.IdempotencyKey)<8||len(v.IdempotencyKey)>100||len(v.Recipient)==0||len(v.Recipient)>8192||!json.Valid(v.Recipient) {
  writeError(w,400,"validation_error","Valid source account, EUR amount, recipient and idempotency key are required");return
 }
 var recipient map[string]any
 if err:=json.Unmarshal(v.Recipient,&recipient);err!=nil||len(recipient)==0 {writeError(w,400,"validation_error","Recipient must be an object");return}
 canonical,_:=json.Marshal(v)
 fingerprint:=sha256.Sum256(canonical)
 var id,status string
 err:=s.db.QueryRow(r.Context(),`
 WITH owned AS (
 SELECT id FROM accounts WHERE id=$1 AND user_id=$2 AND currency='EUR' AND status='active'
 ), inserted AS (
 INSERT INTO payment_instructions(user_id,source_account_id,rail,amount_minor,currency,recipient_currency,idempotency_key,request_fingerprint,recipient_snapshot)
 SELECT $2,id,$3,$4,'EUR','EUR',$5,$6,$7::jsonb FROM owned
 ON CONFLICT(user_id,idempotency_key) DO NOTHING RETURNING id,status
 )
 SELECT id::text,status FROM inserted`,v.SourceAccountID,uid(r),v.Rail,v.AmountMinor,v.IdempotencyKey,fingerprint[:],string(v.Recipient)).Scan(&id,&status)
 if err!=nil {
  var existingID,existingStatus string
  var existingHash []byte
  lookup:=s.db.QueryRow(r.Context(),`SELECT id::text,status,request_fingerprint FROM payment_instructions WHERE user_id=$1 AND idempotency_key=$2`,uid(r),v.IdempotencyKey).Scan(&existingID,&existingStatus,&existingHash)
  if lookup==nil {
   if len(existingHash)!=len(fingerprint)||!bytes.Equal(existingHash,fingerprint[:]) {writeError(w,409,"idempotency_conflict","Idempotency key used with another request");return}
   writeJSON(w,200,map[string]any{"id":existingID,"status":existingStatus,"replayed":true,"provider_connected":false});return
  }
  writeError(w,400,"instruction_not_created","Source account unavailable or payment instruction invalid");return
 }
 writeJSON(w,201,map[string]any{"id":id,"status":status,"provider_connected":false,"notice":"Provider stub only: no funds moved or external payment submitted"})
}
func (s *Server) listPaymentInstructions(w http.ResponseWriter,r *http.Request) {
 rows,err:=s.db.Query(r.Context(),`SELECT id::text,rail,amount_minor,currency,status,created_at FROM payment_instructions WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`,uid(r))
 if err!=nil {writeError(w,500,"internal_error","Could not list instructions");return}
 defer rows.Close()
 items:=[]map[string]any{}
 for rows.Next() {var id,rail,currency,status string;var amount int64;var created any
  if err:=rows.Scan(&id,&rail,&amount,&currency,&status,&created);err!=nil {writeError(w,500,"internal_error","Could not read instructions");return}
  items=append(items,map[string]any{"id":id,"rail":rail,"amount_minor":amount,"currency":strings.TrimSpace(currency),"status":status,"created_at":created})
 }
 if rows.Err()!=nil {writeError(w,500,"internal_error","Could not list instructions");return}
 writeJSON(w,200,map[string]any{"items":items})
}
