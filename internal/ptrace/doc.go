// Package ptrace provides a wrapper for the Unix ptrace APIs.
//
// The original code is based on work by Mahmud "hjr265" Ridwan:
// https://github.com/hjr265/ptrace.go
//
// Stackoverflow user Nominal Animal wrote a very helpful and detailed
// summary of how to use ptrace here:
// https://stackoverflow.com/questions/18577956/how-to-use-ptrace-to-get-a-consistent-view-of-multiple-threads
//
// I have reproduced Nominal Animal's post below to provide some
// guidance on using ptrace.
//
//   - q: Can I attach to a specific thread?
//
//   - a: Yes, at least on current kernels.
//
//   - q: Does that mean that single-stepping only steps through that
//     one thread's instructions? Does it stop all the process's
//     threads?
//
//   - a: Yes. It does not stop the other threads, only the attached
//     one.
//
//   - q: Is there a way to step forward only in one single thread but
//     guarantee that the other threads remain stopped?
//
//   - a: Yes. Send SIGSTOP to the process (use waitpid(PID,,WUNTRACED)
//     to wait for the process to be stopped), then PTRACE_ATTACH
//     to every thread in the process. Send SIGCONT (using
//     waitpid(PID,,WCONTINUED) to wait for the process to continue).
//
// Since all threads were stopped when you attached, and attaching
// stops the thread, all threads stay stopped after the SIGCONT
// signal is delivered. You can single-step the threads in any order
// you prefer.
//
// I found this interesting enough to whip up a test case. (Okay,
// actually I suspect nobody will take my word for it anyway,
// so I decided it's better to show proof you can duplicate on
// your own instead.)
//
// My system seems to follow the man 2 ptrace as described in the
// Linux man-pages project, and Kerrisk seems to be pretty good
// at maintaining them in sync with kernel behaviour. In general,
// I much prefer kernel.org sources wrt. the Linux kernel to other
// sources.
//
// Summary:
//
//   - Attaching to the process itself (TID==PID) stops only the
//     original thread, not all threads.
//   - Attaching to a specific thread (using TIDs from /proc/PID/task/)
//     does stop that thread. (In other words, the thread with
//     TID == PID is not special.)
//   - Sending a SIGSTOP to the process will stop all threads, but
//     ptrace() still works absolutely fine.
//   - If you sent a SIGSTOP to the process, do not call
//     ptrace(PTRACE_CONT, TID) before detaching. PTRACE_CONT seems
//     to interfere with the SIGCONT signal.
//   - You can first send a SIGSTOP, then PTRACE_ATTACH, then send
//     SIGCONT, without any issues; the thread will stay stopped (due
//     to the ptrace). In other words, PTRACE_ATTACH and PTRACE_DETACH
//     mix well with SIGSTOP and SIGCONT, without any side effects
//     I could see.
//   - SIGSTOP and SIGCONT affect the entire process, even if you
//     try using tgkill() (or pthread_kill()) to send the signal to
//     a specific thread.
//   - To stop and continue a specific thread, PTHREAD_ATTACH it; to
//     stop and continue all threads of a process, send SIGSTOP and
//     SIGCONT signals to the process, respectively.
package ptrace
