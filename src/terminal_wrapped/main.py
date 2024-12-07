import os
from collections import Counter, defaultdict
from datetime import datetime
from typing import Dict, List, Tuple

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
                    timestamp_str = line.split(":")[1].strip()
                    command = line.split(";", 1)[1].strip()
                    timestamp = datetime.fromtimestamp(int(timestamp_str))
                    commands.append((command, timestamp))
            except (IndexError, ValueError):
                continue
    return commands


def get_special_chars(command: str) -> str:
    """Extract all non-alphanumeric characters from command."""
    return "".join(
        sorted(
            set(char for char in command if not char.isalnum() and not char.isspace())
        )
    )


def calculate_command_complexity(command: str) -> int:
    """Calculate command complexity based on special characters."""
    special_chars = set("|&;()<>[]{}$\\'\"`!#*?")
    return sum(1 for char in command if char in special_chars)


def get_top_commands(
    commands: List[Tuple[str, datetime]], n: int = 10
) -> List[Tuple[str, int]]:
    """Get the most frequently used base commands."""
    base_commands = []
    for cmd, _ in commands:
        base_cmd = cmd.split()[0] if cmd.split() else ""
        base_commands.append(base_cmd)
    return Counter(base_commands).most_common(n)


def get_top_full_commands(
    commands: List[Tuple[str, datetime]], n: int = 10
) -> List[Tuple[str, int]]:
    """Get the most frequently used full commands."""
    return Counter(cmd for cmd, _ in commands).most_common(n)


def get_most_complex_commands(
    commands: List[Tuple[str, datetime]], n: int = 5
) -> List[Tuple[str, int, str]]:
    """Get the most complex commands with their special characters."""
    command_info = []
    for cmd, _ in commands:
        complexity = calculate_command_complexity(cmd)
        special_chars = get_special_chars(cmd)
        command_info.append((cmd, complexity, special_chars))
    return sorted(command_info, key=lambda x: x[1], reverse=True)[:n]


def get_hourly_distribution(commands: List[Tuple[str, datetime]]) -> Dict[int, int]:
    """Get command distribution by hour."""
    hour_counts = defaultdict(int)
    for _, timestamp in commands:
        hour_counts[timestamp.hour] += 1
    return dict(sorted(hour_counts.items()))


def parse_aliases(zshrc_path: str) -> Dict[str, str]:
    """Parse aliases from .zshrc file."""
    aliases = {}
    try:
        with open(zshrc_path, "r", encoding="utf-8") as f:
            for line in f:
                if line.strip().startswith("alias"):
                    parts = line.strip().split("=", 1)
                    if len(parts) == 2:
                        alias = parts[0].replace("alias", "").strip()
                        command = parts[1].strip().strip("'\"")
                        aliases[alias] = command
    except FileNotFoundError:
        return {}
    return aliases


def analyze_alias_usage(
    aliases: Dict[str, str], commands: List[Tuple[str, datetime]]
) -> Dict[str, int]:
    """Analyze how often each alias is used."""
    alias_usage = defaultdict(int)
    for cmd, _ in commands:
        base_cmd = cmd.split()[0]
        if base_cmd in aliases:
            alias_usage[base_cmd] += 1
    return dict(alias_usage)


def format_section_header(title: str) -> str:
    """Format section header with cool colors."""
    return f"\n{Fore.CYAN}={'='*20} {Fore.YELLOW}{title} {Fore.CYAN}{'='*20}{Style.RESET_ALL}\n"


def format_bar(value: int, max_value: int, width: int = 20) -> str:
    """Format a progress bar with given value and maximum."""
    percentage = value / max_value if max_value > 0 else 0
    filled = int(width * percentage)
    return f"{Fore.RED}{'█' * filled}{Style.RESET_ALL}"


def format_time_bar(hour_counts: Dict[int, int], hour: int) -> str:
    """Format a time bar for 24-hour visualization."""
    max_count = max(hour_counts.values())
    intensity = hour_counts.get(hour, 0) / max_count if max_count > 0 else 0
    if intensity > 0.75:
        return "█"
    elif intensity > 0.5:
        return "▓"
    elif intensity > 0.25:
        return "▒"
    elif intensity > 0:
        return "░"
    return " "


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

        # Top base commands
        print(format_section_header("🌟 Your Top Hits (Base Commands) 🌟"))
        top_commands = get_top_commands(commands)
        max_count = max(count for _, count in top_commands)
        for cmd, count in top_commands:
            print(
                f"{Fore.YELLOW}{cmd:<15}{Fore.WHITE} {count:>5} │ {format_bar(count, max_count)}"
            )

        # Top full commands
        print(format_section_header("🎼 Your Top Hits (Full Commands) 🎼"))
        top_full_commands = get_top_full_commands(commands)
        max_full_count = max(count for _, count in top_full_commands)
        for cmd, count in top_full_commands:
            truncated_cmd = cmd[:50] + "..." if len(cmd) > 50 else cmd
            print(
                f"{Fore.YELLOW}{truncated_cmd:<53}{Fore.WHITE} {count:>5} │ {format_bar(count, max_full_count)}"
            )

        # Most complex commands
        print(format_section_header("🎸 Your Command Symphonies 🎸"))
        complex_commands = get_most_complex_commands(commands)
        max_complexity = max(complexity for _, complexity, _ in complex_commands)
        for cmd, complexity, special_chars in complex_commands:
            truncated_cmd = cmd[:50] + "..." if len(cmd) > 50 else cmd
            print(
                f"{Fore.YELLOW}Complexity: {complexity:<3} │ {format_bar(complexity, max_complexity)}"
            )
            print(f"{Fore.BLUE}Special chars: {Fore.WHITE}{special_chars}")
            print(f"{Fore.CYAN}{truncated_cmd}{Style.RESET_ALL}\n")

        # Time analysis
        print(format_section_header("⏰ Your Terminal Prime Time ⏰"))
        hour_counts = get_hourly_distribution(commands)

        # 24-hour activity visualization
        print(f"{Fore.WHITE}Hour  │ Activity")
        print(f"───────┼{'─' * 24}")
        for hour in range(24):
            bar = "".join(
                Fore.GREEN + format_time_bar(hour_counts, h) + Style.RESET_ALL
                for h in range(24)
            )
            count = hour_counts.get(hour, 0)
            print(f"{hour:02d}:00 │ {bar} {count:>4}")

        # Alias analysis
        print(format_section_header("🎹 Your Alias Symphony 🎹"))
        zshrc_path = os.path.expanduser("~/.zshrc")
        aliases = parse_aliases(zshrc_path)
        alias_usage = analyze_alias_usage(aliases, commands)

        # Most used aliases
        print(f"{Fore.YELLOW}Most Used Aliases:")
        for alias, count in sorted(
            alias_usage.items(), key=lambda x: x[1], reverse=True
        )[:5]:
            print(
                f"{Fore.WHITE}{alias:<15} → {Fore.CYAN}{aliases[alias]:<30} {Fore.WHITE}({count} uses)"
            )

        # Unused aliases
        print(f"\n{Fore.YELLOW}Neglected Aliases:")
        unused = set(aliases.keys()) - set(alias_usage.keys())
        for alias in list(unused)[:5]:
            print(f"{Fore.WHITE}{alias:<15} → {Fore.CYAN}{aliases[alias]}")

    except Exception as e:
        print(f"{Fore.RED}Error: {e}{Style.RESET_ALL}")


if __name__ == "__main__":
    main()
