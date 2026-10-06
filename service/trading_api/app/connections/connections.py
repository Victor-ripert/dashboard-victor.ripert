from datetime import datetime, timezone
from uuid import uuid4
from sqlalchemy import Boolean, DateTime, String, Text, select
from sqlalchemy.orm import Mapped, Session, mapped_column
from app.db import Base


class BrokerConnectionRow(Base):
    __tablename__ = "broker_connections"

    id: Mapped[str] = mapped_column(
        String(36),
        primary_key=True,
        default=lambda: str(uuid4()),
    )

    user_id: Mapped[str] = mapped_column(
        String(100),
        unique=True,
        index=True,
        nullable=False,
    )

    broker: Mapped[str] = mapped_column(
        String(30),
        default="alpaca",
        nullable=False,
    )

    environment: Mapped[str] = mapped_column(
        String(10),
        nullable=False,
    )

    encrypted_access_token: Mapped[str] = mapped_column(
        Text,
        nullable=False,
    )

    alpaca_account_id: Mapped[str | None] = mapped_column(
        String(100),
        nullable=True,
    )

    active: Mapped[bool] = mapped_column(
        Boolean,
        default=True,
        nullable=False,
    )

    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
        onupdate=lambda: datetime.now(timezone.utc),
    )


class ConnectionRepository:
    def __init__(self, db: Session):
        self.db = db

    def get_users(
        self,
        user_id: str,
    ) -> BrokerConnectionRow | None:
        statement = select(BrokerConnectionRow).where(
            BrokerConnectionRow.user_id == user_id,
            BrokerConnectionRow.active.is_(True),
        )

        return self.db.scalar(statement)

    def save(
        self,
        user_id: str,
        environment: str,
        encrypted_access_token: str,
        alpaca_account_id: str | None,
    ) -> BrokerConnectionRow:
        connection = self.get_users(user_id)

        if connection is None:
            connection = BrokerConnectionRow(user_id=user_id)
            self.db.add(connection)

        connection.environment = environment
        connection.encrypted_access_token = encrypted_access_token
        connection.alpaca_account_id = alpaca_account_id
        connection.active = True

        self.db.commit()
        self.db.refresh(connection)

        return connection

    def disconnect(self, user_id: str) -> bool:
        connection = self.get_users(user_id)

        if connection is None:
            return False

        connection.active = False
        self.db.commit()

        return True