import { redirect } from 'next/navigation';

export default function UserLoginRedirect() {
  redirect('/login');
}
