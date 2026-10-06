import { Link } from "react-router-dom";
import AuthForm from "../components/auth/AuthForm";
import { useAuth } from "../auth/useAuth";

export default function RegisterPage() {
  const { register } = useAuth();

  return (
    <AuthForm
      title="Create an account"
      submitLabel="Create account"
      passwordAutoComplete="new-password"
      passwordHint="Between 8 and 72 characters."
      minPasswordLength={8}
      onSubmit={register}
      footer={<>Already have an account? <Link to="/login" className="font-semibold text-sky-400 hover:underline">Log in</Link></>}
    />
  );
}
