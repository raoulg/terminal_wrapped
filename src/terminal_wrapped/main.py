#!/usr/bin/env python3
import os
from collections import Counter, defaultdict
from datetime import datetime
from typing import List, Tuple

from colorama import Fore, Style, init

# Initialize colorama
init()

ASCII_HEADER = """
╔════════════════════════════════════════════════════════════════╗
║  ▄▄▄█████▓▓█████  ██▀███   ███▄ ▄███▓ ██▓ ███▄    █  ▄▄▄      ║
║  ▓  ██▒ ▓▒▓█   ▀ ▓██ ▒ ██▒▓██▒▀█▀ ██▒▓██▒ ██ ▀█   █ ▒████▄    ║
║  ▒ ▓██░ ▒░▒███   ▓██ ░▄█ ▒▓██    ▓██░▒██▒▓██  ▀█ ██▒▒██  ▀█▄  ║
║  ░ ▓██▓ ░ ▒▓█  ▄ ▒██▀▀█▄  ▒██    ▒██ ░██░▓██▒  ▐▌██▒░██▄▄▄▄██ ║
║    ▒██▒ ░ ░▒████▒░██▓ ▒██▒▒██▒   ░██▒░██░▒██░   ▓██░ ▓█   ▓██▒║
║    ▒ ░░   ░░ ▒░ ░░ ▒▓ ░▒▓░░ ▒░   ░  ░░▓  ░ ▒░   ▒ ▒  ▒▒   ▓▒█░║
║      ░     ░ ░  ░  ░▒ ░ ▒░░  ░      ░ ▒ ░░ ░░   ░ ▒░  ▒   ▒▒ ░║
║    ░         ░     ░░   ░ ░      ░    ▒ ░   ░   ░ ░   ░   ▒   ║
║              ░  ░   ░            ░    ░           ░       ░  ░║
║                                                               ║
║                     W R A P P E D   2 0 2 4                   ║
╚════════════════════════════════════════════════════════════════╝
"""


def get_history_file() -> str:
    """Get the path to the shell history file."""
    shell = os.environ.get("SHELL", "").split("/")[-1]
    home = os.path.expanduser("~")

    if shell == "zsh":
        return os.path.join(home, ".zsh_history")
    elif shell == "bash":
        return os.path.join(home, ".bash_history")
    else:
        raise ValueError(f"Unsupported shell: {shell}")


def parse_zsh_history(history_file: str) -> List[Tuple[str, datetime]]:
    """Parse ZSH history file with timestamps."""
    commands = []
    with open(history_file, "r", encoding="utf-8", errors="ignore") as f:
        for line in f:
            try:
                if line.startswith(": "):
                    # ZSH timestamp format: ': 1234567890:0;command'
                    timestamp_str = line.split(":")[1].strip()
                    command = line.split(";", 1)[1].strip()
                    timestamp = datetime.fromtimestamp(int(timestamp_str))
                    commands.append((command, timestamp))
            except (IndexError, ValueError):
                continue
    return commands


def calculate_command_complexity(command: str) -> int:
    """Calculate command complexity based on special characters."""
    special_chars = set("|&;()<>[]{}$\\'\"`!#*?")
    return sum(1 for char in command if char in special_chars)


def get_top_commands(
    commands: List[Tuple[str, datetime]], n: int = 10
) -> List[Tuple[str, int]]:
    """Get the most frequently used commands."""
    base_commands = []
    for cmd, _ in commands:
        base_cmd = cmd.split()[0] if cmd.split() else ""
        base_commands.append(base_cmd)
    return Counter(base_commands).most_common(n)


def get_most_complex_commands(
    commands: List[Tuple[str, datetime]], n: int = 5
) -> List[Tuple[str, int]]:
    """Get the most complex commands based on special character count."""
    return sorted(
        [(cmd, calculate_command_complexity(cmd)) for cmd, _ in commands],
        key=lambda x: x[1],
        reverse=True,
    )[:n]


def get_busiest_hour(commands: List[Tuple[str, datetime]]) -> Tuple[int, int]:
    """Get the hour with most commands."""
    hour_counts = defaultdict(int)
    for _, timestamp in commands:
        hour_counts[timestamp.hour] += 1
    return max(hour_counts.items(), key=lambda x: x[1])


def format_section_header(title: str) -> str:
    """Format section header with cool colors."""
    return f"\n{Fore.CYAN}={'='*20} {Fore.YELLOW}{title} {Fore.CYAN}{'='*20}{Style.RESET_ALL}\n"


def main():
    print(Fore.GREEN + ASCII_HEADER + Style.RESET_ALL)

    try:
        history_file = get_history_file()
        commands = parse_zsh_history(history_file)

        # Basic stats
        total_commands = len(commands)
        unique_commands = len(set(cmd for cmd, _ in commands))

        print(format_section_header("🎵 Your Terminal Rhythm 🎵"))
        print(f"{Fore.MAGENTA}Total commands: {Fore.WHITE}{total_commands}")
        print(f"{Fore.MAGENTA}Unique commands: {Fore.WHITE}{unique_commands}")

        # Top commands
        print(format_section_header("🌟 Your Top Hits 🌟"))
        for cmd, count in get_top_commands(commands):
            percentage = (count / total_commands) * 100
            bar_length = int(percentage / 2)
            print(
                f"{Fore.YELLOW}{cmd:<15}{Fore.WHITE} {count:>5} │ {Fore.RED}{'█' * bar_length}{Style.RESET_ALL}"
            )

        # Most complex commands
        print(format_section_header("🎸 Your Command Symphonies 🎸"))
        complex_commands = get_most_complex_commands(commands)
        for cmd, complexity in complex_commands:
            print(f"{Fore.BLUE}Complexity: {Fore.WHITE}{complexity}")
            print(f"{Fore.CYAN}{cmd}{Style.RESET_ALL}\n")

        # Time analysis
        print(format_section_header("⏰ Your Terminal Prime Time ⏰"))
        busiest_hour, count = get_busiest_hour(commands)
        print(
            f"{Fore.MAGENTA}Most active hour: {Fore.WHITE}{busiest_hour:02d}:00 ({count} commands)"
        )

        # Late night warrior
        late_night_commands = sum(
            1 for _, ts in commands if ts.hour >= 0 and ts.hour < 5
        )
        if late_night_commands > 0:
            print(
                f"{Fore.RED}Night Owl Alert! {Fore.WHITE}{late_night_commands} commands between midnight and 5 AM!"
            )

    except Exception as e:
        print(f"{Fore.RED}Error: {e}{Style.RESET_ALL}")


if __name__ == "__main__":
    main()
