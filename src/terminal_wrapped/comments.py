def get_command_count_comment(total_commands: int) -> str:
    """Get a witty comment based on the number of commands."""
    if total_commands < 20:
        return "🐣 Just hatched! Everyone starts somewhere..."
    elif total_commands < 100:
        return "🌱 Growing terminal-ist! The journey of a thousand commands begins with a single keystroke."
    elif total_commands < 500:
        return "🚀 Houston, we have lift-off! Your terminal journey is taking shape."
    elif total_commands < 1000:
        return "🎮 Terminal warrior in training! Vi would be proud."
    elif total_commands < 5000:
        return "⚡ Power user alert! Your keyboard is probably glowing."
    elif total_commands < 15000:
        return "🔥 Terminal virtuoso! Your fingers probably dream in bash."
    else:
        return "🧙‍♂️ You're basically Gandalf the Grey of the terminal! 'YOU SHALL NOT GUI!'"


def get_top_command_comment(cmd: str) -> str:
    """Get a witty comment based on the most used command."""
    comments = {
        "git": [
            "🤔 Ah, a fellow time traveler! Making history, one commit at a time.",
            "Someone's got commitment issues... in a good way!",
            "Git happens, but you're handling it like a pro!",
        ],
        "cd": [
            "🏃‍♂️ Can't sit still, can you? A true directory explorer!",
            "Home is where the ~/ is!",
            "Walking directory trees keeps you fit!",
        ],
        "nvim": [
            "🎹 A vim virtuoso! Escape key is probably worn out.",
            "Modal editing: because why use a mouse when you have 10 fingers?",
            ":wq is your signature move!",
        ],
        "ls": [
            "👀 Someone likes to know what's going on!",
            "Directory detective at work!",
            "List first, ask questions later!",
        ],
        "rm": [
            "☠️ Living dangerously, I see! Hope you have backups...",
            "The delete key wasn't permanent enough for you, huh?",
            "Making space, one file at a time!",
        ],
        "make": [
            "🏗️ Building things, breaking things, it's all in a day's work!",
            "Make it work, make it right, make it fast!",
            "You're a maker, not a faker!",
        ],
    }
    default_comments = [
        "This command must feel like home by now!",
        "Your fingers could type this in their sleep!",
        "Your favorite dance move in the terminal!",
    ]
    cmd_base = cmd.split()[0]
    return comments.get(cmd_base, default_comments)[0]


def get_complexity_comment(complexity: int) -> str:
    """Get a comment based on command complexity."""
    if complexity < 5:
        return "💫 Simple and sweet!"
    elif complexity < 10:
        return "🎭 Getting fancy there!"
    elif complexity < 15:
        return "🎪 Now we're juggling with characters!"
    elif complexity < 20:
        return "🌟 Regular expression royalty!"
    else:
        return "🎯 Maximum complexity achieved! Perl would be proud!"


def get_hour_comment(hour: int, count: int) -> str:
    """Get a comment based on the hour and activity."""
    if hour < 5 and count > 0:
        return "🦉 Night owl alert! Bug hunting in the dark?"
    elif 5 <= hour < 8 and count > 0:
        return "🌅 Early bird gets the code merged!"
    elif 8 <= hour < 12:
        return "☕ Coffee-powered coding session!"
    elif 12 <= hour < 14:
        return "🍜 Lunch break coding warrior!"
    elif 14 <= hour < 18:
        return "⚡ Peak productivity power hour!"
    elif 18 <= hour < 22:
        return "🌙 Evening excellence!"
    else:
        return "🌚 Midnight commander!"
