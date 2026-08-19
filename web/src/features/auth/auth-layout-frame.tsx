/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
type AuthLayoutFrameProps = {
  brand: React.ReactNode
  children: React.ReactNode
}

export function AuthLayoutFrame(props: AuthLayoutFrameProps) {
  return (
    <main
      data-slot='auth-page'
      className='relative isolate min-h-svh overflow-x-hidden'
    >
      <div
        data-slot='auth-background'
        aria-hidden='true'
        className='auth-swirl-background pointer-events-none fixed inset-0 -z-10'
      />
      <div className='container flex min-h-svh items-center justify-center px-4 py-6 sm:py-10'>
        <section
          data-slot='auth-card'
          className='auth-light-surface flex w-full max-w-[520px] flex-col justify-center space-y-8 rounded-[8px] border border-white/70 px-6 py-8 shadow-[0_24px_80px_-28px_rgba(15,23,42,0.38)] backdrop-blur-xl sm:px-10 sm:py-10'
        >
          {props.brand}
          {props.children}
        </section>
      </div>
    </main>
  )
}
