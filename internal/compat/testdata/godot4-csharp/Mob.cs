using Godot;

[GlobalClass]
public partial class Mob : Node2D
{
    public void Chase(Vector2 target) => Position = Position.MoveToward(target, 10);
}
