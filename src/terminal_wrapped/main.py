import os
import textwrap
from collections import Counter, defaultdict
from datetime import datetime
from typing import Dict, List, Tuple

from colorama import Fore, Style, init

from terminal_wrapped.comments import (
    get_command_count_comment,
    get_complexity_comment,
    get_top_command_comment,
)

# Initialize colorama
init()

ASCII_HEADER = """
╔════════════════════════════════════════════════════════════════╗
║  ▄▄▄█████▓▓█████  ██▀███   ███▄ ▄███▓ ██▓ ███▄    █  ▄▄▄       ║
║  ▓  ██▒ ▓▒▓█   ▀ ▓██ ▒ ██▒▓██▒▀█▀ ██▒▓██▒ ██ ▀█   █ ▒████▄     ║
║  ▒ ▓██░ ▒░▒███   ▓██ ░▄█ ▒▓██    ▓██░▒██▒▓██  ▀█ ██▒▒██  ▀█▄   ║
║  ░ ▓██▓ ░ ▒▓█  ▄ ▒██▀▀█▄  ▒██    ▒██ ░██░▓██▒  ▐▌██▒░██▄▄▄▄██  ║
║    ▒██▒ ░ ░▒████▒░██▓ ▒██▒▒██▒   ░██▒░██░▒██░   ▓██░ ▓█   ▓██▒ ║
║    ▒ ░░   ░░ ▒░ ░░ ▒▓ ░▒▓░░ ▒░   ░  ░░▓  ░ ▒░   ▒ ▒  ▒▒   ▓▒█░ ║
║      ░     ░ ░  ░  ░▒ ░ ▒░░  ░      ░ ▒ ░░ ░░   ░ ▒░  ▒   ▒▒ ░ ║
║    ░         ░     ░░   ░ ░      ░    ▒ ░   ░   ░ ░   ░   ▒    ║
║              ░  ░   ░            ░    ░           ░       ░  ░ ║
║                                                                ║
║                     W R A P P E D   2 0 2 4                    ║
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


def print_header():
    """Print the ASCII art header."""
    print(Fore.GREEN + ASCII_HEADER + Style.RESET_ALL)


def print_terminal_rhythm(total_commands: int, unique_commands: int):
    """Print basic command statistics."""
    print(format_section_header("🎵 Your Terminal Rhythm 🎵"))
    print(f"{Fore.MAGENTA}Total commands: {Fore.WHITE}{total_commands}")
    print(f"{Fore.MAGENTA}Unique commands: {Fore.WHITE}{unique_commands}")
    print(
        f"\n{Fore.YELLOW}{get_command_count_comment(total_commands)}{Style.RESET_ALL}"
    )


def print_base_commands(commands: List[Tuple[str, datetime]], aliases: Dict[str, str]):
    """Print analysis of true base commands (excluding aliases)."""
    print(format_section_header("🌟 Your Top Raw Commands 🌟"))

    # Count only non-alias base commands
    base_counts = defaultdict(int)
    for cmd, _ in commands:
        base_cmd = cmd.split()[0]
        if base_cmd not in aliases:  # Only count if not an alias
            base_counts[base_cmd] += 1

    base_commands = sorted(base_counts.items(), key=lambda x: x[1], reverse=True)[:10]
    if not base_commands:
        print(
            f"{Fore.YELLOW}Wow, you really love your aliases! No raw commands found.{Style.RESET_ALL}"
        )
        return

    max_base_count = max(count for _, count in base_commands)

    for cmd, count in base_commands:
        print(
            f"{Fore.YELLOW}{cmd:<15}{Fore.WHITE} {count:>5} │ {format_bar(count, max_base_count)}"
        )


def print_full_commands(commands: List[Tuple[str, datetime]], aliases: Dict[str, str]):
    """Print analysis of full commands with alias expansion."""
    print(format_section_header("🎼 Your Alias Artistry 🎼"))

    # Count all commands
    command_counts = defaultdict(int)
    for cmd, _ in commands:
        base_cmd = cmd.split()[0]
        if base_cmd in aliases:
            # Use the full expanded command as the key
            expanded = cmd.replace(base_cmd, aliases[base_cmd], 1)
            command_counts[expanded] += 1
        else:
            command_counts[cmd] += 1

    full_commands = sorted(command_counts.items(), key=lambda x: x[1], reverse=True)[
        :10
    ]
    max_full_count = max(count for _, count in full_commands)

    # Display commands
    for cmd, count in full_commands:
        # For expanded commands, show original alias in parentheses
        base_cmd = cmd.split()[0]
        for alias, expansion in aliases.items():
            if cmd.startswith(expansion):
                cmd = f"{cmd} ({alias})"
                break

        print(
            f"{Fore.YELLOW}{cmd:<50}{Fore.WHITE} {count:>5} │ {format_bar(count, max_full_count)}"
        )

    print(
        f"\n{Fore.YELLOW}These are your favorite aliases and commands.{Style.RESET_ALL}"
    )
    print(
        f"{Fore.YELLOW}{get_top_command_comment(full_commands[0][0].split()[0])}{Style.RESET_ALL}"
    )


def print_command_complexity(commands: List[Tuple[str, datetime]]):
    """Print analysis of command complexity."""
    print(format_section_header("🎸 Your Command Symphonies 🎸"))
    complex_commands = get_most_complex_commands(commands)
    max_complexity = max(complexity for _, complexity, _ in complex_commands)

    max_special_chars = ""
    for cmd, complexity, _ in complex_commands:
        special_chars = "".join(c for c in cmd if not c.isalnum() and not c.isspace())
        if complexity == max_complexity:
            max_special_chars = special_chars

        print(
            f"{Fore.YELLOW}Complexity: {complexity:<3} │ {format_bar(complexity, max_complexity)}"
        )
        print(f"{Fore.BLUE}Special characters: {Fore.WHITE}{special_chars}")
        wrapped_cmd = textwrap.fill(cmd, width=70, subsequent_indent="    ")
        print(f"{Fore.CYAN}{wrapped_cmd}{Style.RESET_ALL}\n")

    print(
        f"{Fore.YELLOW}Maximum complexity score: {max_complexity}! Your most complex command uses {len(max_special_chars)} unique special characters.{Style.RESET_ALL}"
    )
    print(f"{Fore.GREEN}{get_complexity_comment(max_complexity)}{Style.RESET_ALL}")


def print_time_analysis(commands: List[Tuple[str, datetime]]):
    """Print analysis of command timing with daily overview and insights."""
    print(format_section_header("⏰ Your Terminal Prime Time ⏰"))
    hour_counts = get_hourly_distribution(commands)
    max_hour_count = max(hour_counts.values())

    # Configure visualization width
    hour_width = 2  # Width per hour block
    total_width = hour_width * 24  # Total width for 24 hours

    # Print time markers - ensure proper spacing between hours
    print("Daily Activity Pattern:")
    times = ["00:00", "06:00", "12:00", "18:00", "24:00"]
    time_line = ""
    current_pos = 0

    for i, time in enumerate(times):
        target_pos = (i * total_width) // 4
        padding = target_pos - current_pos
        time_line += " " * padding + time
        current_pos = target_pos + len(time)

    # Create divider line with same positioning
    divider_line = ""
    current_pos = 0
    for i in range(len(times)):
        target_pos = (i * total_width) // 4
        padding = target_pos - current_pos
        divider_line += " " * padding + "│"
        current_pos = target_pos + 1

    print(time_line)
    print(divider_line)

    # Generate the activity visualization
    activity_line = ""
    for hour in range(24):
        count = hour_counts.get(hour, 0)
        intensity = count / max_hour_count if max_hour_count > 0 else 0
        if intensity > 0.75:
            block = Fore.GREEN + "██"
        elif intensity > 0.5:
            block = Fore.YELLOW + "██"
        elif intensity > 0.25:
            block = Fore.RED + "██"
        elif intensity > 0:
            block = Fore.BLUE + "██"
        else:
            block = Fore.WHITE + "··"
        activity_line += block

    print(f"{activity_line}{Style.RESET_ALL}\n")

    # Print detailed hourly breakdown
    print(f"{Fore.WHITE}Hour  │ Commands")
    print(f"───────┼{'─' * 50}")

    for hour in range(24):
        count = hour_counts.get(hour, 0)
        intensity = count / max_hour_count if max_hour_count > 0 else 0
        bar_length = int(intensity * 40)
        bar = ("█" * bar_length).ljust(40)
        print(f"{hour:02d}:00 │ {bar} {count:>4}")

    # Add insights about the day
    print(f"\n{Fore.YELLOW}Daily Insights:{Style.RESET_ALL}")

    # Find first active hour between 5:00 and 23:59
    start_hour = min(
        (h for h, c in hour_counts.items() if c > 0 and 5 <= h < 24), default=None
    )
    if not start_hour:
        start_hour = min((h for h, c in hour_counts.items() if c > 0), default=0)

    most_active_hour = max(hour_counts.items(), key=lambda x: x[1])[0]
    late_night_count = sum(hour_counts.get(h, 0) for h in range(0, 5))

    # Get most used command for specific hours
    def get_hour_top_command(hour: int) -> Tuple[str, int]:
        hour_commands = [cmd for cmd, ts in commands if ts.hour == hour]
        if not hour_commands:
            return ("", 0)
        return Counter(hour_commands).most_common(1)[0]

    print(
        f"🌅 Day starts at {start_hour:02d}:00 with {hour_counts[start_hour]} commands"
    )
    top_morning_cmd, count = get_hour_top_command(start_hour)
    if top_morning_cmd:
        print(f"   First order of business: {top_morning_cmd}")

    print(
        f"⚡ Peak activity at {most_active_hour:02d}:00 with {hour_counts[most_active_hour]} commands"
    )
    top_peak_cmd, count = get_hour_top_command(most_active_hour)
    if top_peak_cmd:
        print(f"   Favorite command: {top_peak_cmd}")

    if late_night_count > 0:
        print(f"🦉 {late_night_count} commands between midnight and 5 AM")
        top_night_cmd, count = get_hour_top_command(
            max(h for h in range(5) if hour_counts.get(h, 0) > 0)
        )
        if top_night_cmd:
            print(f"   Late night special: {top_night_cmd}")


def print_alias_analysis(commands: List[Tuple[str, datetime]], aliases: Dict[str, str]):
    """Print analysis of alias usage."""
    print(format_section_header("🎹 Your Alias Symphony 🎹"))

    if not aliases:
        print(
            f"{Fore.YELLOW}No aliases found! Living life on hard mode, eh?{Style.RESET_ALL}"
        )
        return

    print(f"{Fore.YELLOW}Most Used Aliases:")
    alias_usage = analyze_alias_usage(aliases, commands)
    max_alias_count = max(alias_usage.values()) if alias_usage else 0

    # Find longest alias for proper formatting
    max_alias_len = max(len(alias) for alias in aliases.keys())
    # Calculate position for count and bar
    count_position = 60  # Adjust this value to change alignment

    for alias, count in sorted(alias_usage.items(), key=lambda x: x[1], reverse=True)[
        :5
    ]:
        expansion = aliases[alias]
        # Start with alias and expansion
        line = f"{Fore.WHITE}{alias:<{max_alias_len}} → {Fore.CYAN}{expansion}"
        # Pad to fixed position for count
        padding = count_position - len(alias) - 3 - len(expansion)  # -3 for " → "
        if padding > 0:
            line += " " * padding
        # Add count and bar
        line += f"{Fore.WHITE}{count:>4} │ {format_bar(count, max_alias_count)}"
        print(line)

    print(f"\n{Fore.YELLOW}Neglected Aliases (Show them some ❤️):")
    unused = set(aliases.keys()) - set(alias_usage.keys())
    for alias in list(unused)[:5]:
        print(f"{Fore.WHITE}{alias:<{max_alias_len}} → {Fore.CYAN}{aliases[alias]}")


def clear_screen():
    os.system("cls" if os.name == "nt" else "clear")


def print_welcome():
    clear_screen()
    print(f"{Fore.GREEN}{ASCII_HEADER}{Style.RESET_ALL}")
    print(f"\n{Fore.YELLOW}Welcome to your Terminal Time Machine! 🚀{Style.RESET_ALL}")
    print("\nLet's explore your command line journey through 2024!")
    print("\nUse arrow keys or '<>' to navigate, space/enter to proceed")


def generate_wrapped_report(
    commands: List[Tuple[str, datetime]],
    aliases: Dict[str, str],
    total_commands: int,
    unique_commands: int,
):
    sections = [
        (
            "Terminal Rhythm",
            lambda: print_terminal_rhythm(total_commands, unique_commands),
        ),
        ("Base Commands", lambda: print_base_commands(commands, aliases)),
        ("Command Lines", lambda: print_full_commands(commands, aliases)),
        ("Command Symphonies", lambda: print_command_complexity(commands)),
        ("Time Analysis", lambda: print_time_analysis(commands)),
        ("Alias Symphony", lambda: print_alias_analysis(commands, aliases)),
    ]

    current_section = 0
    print_welcome()

    while True:
        clear_screen()
        print(f"{Fore.GREEN}{ASCII_HEADER}{Style.RESET_ALL}")
        sections[current_section][1]()
        print(
            f"\n{Fore.BLUE}[{current_section + 1}/{len(sections)}] Use n/p for next/previous, q to quit{Style.RESET_ALL}"
        )

        command = input().lower()
        if command == "n":
            current_section += 1
        elif command == "p":
            current_section -= 1
        elif command == "q":
            break

        current_section = max(0, min(current_section, len(sections)))

        if current_section == len(sections):
            clear_screen()
            print(f"\n{Fore.GREEN}Thanks for the memories! 🎉{Style.RESET_ALL}")
            print(f"\n{Fore.YELLOW}Happy coding! May your commits be clean")
            print("and your bugs be few! 🐛✨{Style.RESET_ALL}")
            break


def main():
    try:
        history_file = get_history_file()
        zshrc_path = os.path.expanduser("~/.zshrc")

        commands = parse_zsh_history(history_file)
        aliases = parse_aliases(zshrc_path)
        total_commands = len(commands)
        unique_commands = len(set(cmd for cmd, _ in commands))

        generate_wrapped_report(commands, aliases, total_commands, unique_commands)

    except Exception as e:
        print(f"{Fore.RED}Error: {e}{Style.RESET_ALL}")


if __name__ == "__main__":
    main()
